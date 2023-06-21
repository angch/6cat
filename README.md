# 6cat

cat replacement that converts images into sixel images on supported terminals. If it's not an
image, the contents will be dumped as-is.

For fast installation on random remote comptuers with 0 dependencies, so you can check
on images fast, e.g. image files stored on server.

Majority of code is based off [github.com/mattn/go-sixel](github.com/mattn/go-sixel)

Note: vscode's terminal has experimental sixel support:
[https://code.visualstudio.com/updates/v1_79#_images-in-the-terminal](https://code.visualstudio.com/updates/v1_79#_images-in-the-terminal)

## Compile a statically linked binary

```CGO_ENABLED=0 go build -ldflags "-extldflags '-static' -s -w" -o 6cat```

Prefix `GOOS=linux GOARCH=amd64` if required. Then copy or somehow distribute that one file to your server.

## Notes

Not going to name this cat6, because it confuses googling.
