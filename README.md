# gncdu

gncdu implements [NCurses Disk Usage](https://dev.yorhel.nl/ncdu)(ncdu) with golang, and is at least twice faster as ncdu.

## Install

### Binaries

macOS (Apple Silicon)

    wget -O /usr/local/bin/gncdu https://github.com/bastengao/gncdu/releases/download/v0.7.0/gncdu-darwin-arm64 && chmod +x /usr/local/bin/gncdu

Linux (amd64)

    wget -O /usr/local/bin/gncdu https://github.com/bastengao/gncdu/releases/download/v0.7.0/gncdu-linux-amd64 && chmod +x /usr/local/bin/gncdu

Or download executable file from [releases](https://github.com/bastengao/gncdu/releases) page.

### Install from source

    go install github.com/bastengao/gncdu@latest

## Usage

Basic usage:

    gncdu [options] [directory]

Options:
- `-c` number: Set the number of concurrent scanners (default is number of CPU cores)
- `-t` number: Set threshold in MB for small files grouping (0 means no grouping)
- `--help`: Show help information

Key bindings:
- `↑`, `↓`: Navigate through files/directories
- `Enter`: Enter into directory
- `Backspace`: Go back to parent directory
- `d`: Delete selected file/directory (with confirmation)
- `?`: Show help
- `Ctrl+C`: Exit program

Examples:

    gncdu -c 4 -t 1000 /home/user

![screenshot](http://bastengao.com/images/others/gncdu-screenshot-v0.7.0.png)
