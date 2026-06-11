package main

import (
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// graphicsSupport records which inline-image protocols the active terminal
// understands. It is discovered by querying the terminal itself rather than
// guessing from $TERM, so it works through tmux/ssh and with terminals we've
// never heard of.
type graphicsSupport struct {
	sixel bool
	kitty bool
}

var (
	detectOnce sync.Once
	detected   graphicsSupport
)

// detectGraphics returns the terminal's image-protocol support, querying the
// terminal at most once per process.
func detectGraphics() graphicsSupport {
	detectOnce.Do(func() { detected = queryTerminal() })
	return detected
}

// queryTerminal asks the terminal what it supports by writing two queries in a
// single burst and reading the replies:
//
//   - A kitty graphics query (APC _G…a=q…). Terminals that speak the kitty
//     graphics protocol answer with an APC string containing "_G…;OK".
//   - A Primary Device Attributes request (CSI c / DA1). EVERY terminal answers
//     this, and sixel-capable ones include attribute "4" in the reply
//     (e.g. ESC [ ? 62 ; 4 ; … c).
//
// Because DA1 is sent last and is always answered, its reply reliably
// terminates our read even when the kitty query is silently ignored.
func queryTerminal() graphicsSupport {
	// No point emitting graphics if stdout isn't a terminal (piped/redirected).
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return graphicsSupport{}
	}

	// Talk to the controlling terminal directly: stdin may be the image data
	// (e.g. `cat foo.png | 6cat -`), so it can't carry the query replies.
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fallbackDetect()
	}
	defer tty.Close()

	fd := int(tty.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fallbackDetect()
	}
	defer term.Restore(fd, oldState)

	const query = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\" + // kitty graphics query
		"\x1b[c" // DA1 — always answered, acts as terminator

	if _, err := tty.WriteString(query); err != nil {
		return fallbackDetect()
	}

	resp := readResponse(tty, 250*time.Millisecond)

	return graphicsSupport{
		sixel: hasSixelAttr(resp),
		kitty: strings.Contains(resp, "_G"),
	}
}

// readResponse reads from the terminal until the DA1 reply terminator ('c') is
// seen or the timeout elapses. A short-lived goroutine does the blocking read;
// closing the tty (deferred by the caller) unblocks it afterwards.
func readResponse(tty *os.File, timeout time.Duration) string {
	ch := make(chan byte, 256)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := tty.Read(buf)
			if n > 0 {
				ch <- buf[0]
			}
			if err != nil {
				return
			}
		}
	}()

	var sb strings.Builder
	deadline := time.After(timeout)
	for {
		select {
		case b := <-ch:
			sb.WriteByte(b)
			if b == 'c' { // end of the DA1 reply, which we sent last
				return sb.String()
			}
		case <-deadline:
			return sb.String()
		}
	}
}

// hasSixelAttr reports whether a DA1 reply advertises sixel support, i.e.
// attribute "4" appears in the ESC [ ? … c parameter list.
func hasSixelAttr(resp string) bool {
	_, params, ok := strings.Cut(resp, "[?")
	if !ok {
		return false
	}
	if j := strings.IndexByte(params, 'c'); j >= 0 {
		params = params[:j]
	}
	return slices.Contains(strings.Split(params, ";"), "4")
}

// fallbackDetect is the legacy $TERM heuristic, used only when the terminal
// can't be queried (e.g. /dev/tty is unavailable).
func fallbackDetect() graphicsSupport {
	switch os.Getenv("TERM") {
	case "xterm-kitty", "xterm-ghostty", "alacritty":
		return graphicsSupport{kitty: true}
	default:
		return graphicsSupport{sixel: true}
	}
}
