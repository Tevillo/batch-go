# batch-go
Batch file renamer with a CLI element. Revamp of my Batch repo but hopefully expanded and made in golang

It renames the photos in the current directory to a numbered pattern, e.g.
`IMG_3.jpg, IMG_10.jpg` → `trip-1.jpg, trip-2.jpg`, and can undo the rename.

## Install

Requires Go 1.27 or later.

```sh
git clone https://github.com/Tevillo/batch-go.git
cd batch-go
go install
```

This puts `batch-go` in `$(go env GOPATH)/bin`; make sure that's on your `PATH`.
Use `go build` instead to just build the binary in the current folder.

## Usage

Run it from inside the folder of photos you want to rename:

```sh
cd ~/Pictures/trip
batch-go
```

It picks up `.jpg` files and raw files (`.dng`, `.tif`) and orders them by the
first number in each name, so `IMG_2` comes before `IMG_10`. JPGs and raws are
numbered separately, so `IMG_3.jpg` and `IMG_3.dng` both become `trip-1`.
Numbers are zero-padded to the width of the file count (`trip-01` … `trip-12`).

The default pattern is the folder's name. Then type commands at the prompt:

| Command | Description |
| --- | --- |
| `dry` | Print a dry run of new file names |
| `help`, `h` | Display the help message |
| `list`, `ls` | List the current names of the found photos |
| `quit`, `q` | Exit the program (an empty line also exits) |
| `revert` | Undo the last run, restoring the original file names |
| `run` | Rename the files |
| `set <pattern>` | Set the file name pattern (default: current directory name) |

Example session:

```
set trip
dry
IMG_3.dng -> trip-1.dng
IMG_10.dng -> trip-2.dng
IMG_3.jpg -> trip-1.jpg
IMG_10.jpg -> trip-2.jpg
run
...
DONE!
revert
...
q
```

`revert` only works within the same session; once you quit, the original
names are gone.

## Development

```sh
go test ./...            # run all tests
go test -v ./...         # list each test
go test -cover ./...     # show coverage
```

Tests run in temporary directories and never touch your files.

## License

Apache 2.0. See [LICENSE](LICENSE).
