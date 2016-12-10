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

# Programs

There are several commands included under the `cmd` directory. All
commands accept `-help` to list flags, but are otherwise minimally
documented at present.

Perhaps the most useful is `analyzetak`, which allows you evaluate a
position offline using Taktician's AI:

## analyzetak

A program that reads PTN files and performs AI analysis on the
terminal position.

By default

```
analyzetak FILE.ptn
```

will analyze every position and report Taktician's evaluation and
suggested move. You can also analzye e.g. white's 10th move using:

```
analyzetak -white -move 10 FILE.ptn
```

With no `-move` argument, `analyzetak` will analyze the final position
of the PTN file.

By default, `analyzetak` will search for up to 1m before returning a
final assessment. Use `-limit 2m` to give it more time, or `-depth 5`
to search to a fixed depth.


## `playtak`

A simple interface to play tak on the command line. Try e.g.

```
playtak -white=human -black=minimax:5
```

## taklogger

A bot that connects to playtak.com and logs all games it sees in PTN
format.

## taktician

The AI driver for playtak.com. Can be used via

```
taktician -user USERNAME -pass PASSWORD
```

[tak]: http://cheapass.com/node/215
