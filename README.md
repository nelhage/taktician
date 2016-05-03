# Taktician - A Tak Bot

This repository implements the game of [Tak][tak], including a fairly
strong AI, and support for the playtak.com server.

# Installation

Try

```
go get github.com/nelhage/taktician/cmd/...
```

# Programs

There are several commands included under the `cmd` directory. All
commands accept `-help` to list flags, but are otherwise minimally
documented at present.

## `playtak`

A simple interface to play tak on the command line. Try e.g.

```
playtak -white=human -black=minimax:5
```

## analyzetak

A program that reads PTN files and performs AI analysis on the
terminal position.

```
analyzetak FILE.ptn
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
