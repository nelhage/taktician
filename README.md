# Taktician - A Tak Bot

This repository implements the game of [Tak][tak], including a fairly
strong AI, and support for the playtak.com server.

# Installation

Taktician requires `go1.7` or newer. On OS X, try `brew update && brew
install go`.

Once you have a working `go` installation, you can fetch+install the
below commands using:


```
go get -u github.com/nelhage/taktician/cmd/...
```

Alternately, if you have a checkout of this repository, build+install
it using

```
go install ./cmd/...
```

to install a `taktician` binary into your `$GOPATH/bin` (`~/go/bin` by
default).

# Subcommands

Taktician consists of a single binary, `taktician`, which accepts a
number of subcommands. You can run `taktician -help` to list all
available commands, and `taktician [command] -help` for details on the
options available to an individual command.

Perhaps the most generally useful subcommand is `taktician analyze`,
which allows you evaluate a position offline using Taktician's AI:

## taktician analyze

A command that reads PTN files and performs AI analysis on the
terminal position.

By default

```
taktician analyze FILE.ptn
```

will analyze the final position and report Taktician's evaluation and
suggested move.

You can also analzye e.g. white's 10th move using:

```
taktician analyze -white -move 10 FILE.ptn
```

With `-all`, `taktician analyze` will analyze each position in the PTN
file.

By default, `taktician analyze` will search for up to 1m before
returning a final assessment. Use `-limit 2m` to give it more time, or
`-depth 5` to search to a fixed depth.


## `taktician play`

A simple interface to play tak on the command line. Try e.g.

```
taktician play -white=human -black=minimax:5
```

## `taktician playtak`

The AI driver for playtak.com. Can be used via

```
taktician playtak -user USERNAME -pass PASSWORD
```

[tak]: https://cheapass.com/tak/
