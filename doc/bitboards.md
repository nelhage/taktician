# bitboards for Tak

It is [well-known][chess] in the field of chess AI that one efficient
representation of a chess board uses of 64-bit integers as bit-sets to
represent features of the 8x8 board. In addition to being a very
compact representation, this format allows for all kinds of clever
tricks when computing possible moves and attacks on the board.

[chess]: https://chessprogramming.wikispaces.com/Bitboards

Compared to chess, Tak has the additional complication of
3-dimensional stacks, which are less obviously representable in a
bitwise fashion. This document describes Taktician's approach to
efficiently representing Tak boards using bit-wise representations.

# Stack tops

The top piece in each stack has special significance in Tak. Walls and
Capstones may only be present on top of stacks, and the top piece is
relevant for determining roads, for scoring the game for flat wins,
and for determining control over each stack.

In light of these special properties of the top piece in each stack,
Taktician represents the top of each stack separately from the rest of
the board.

Taktician stores the board-state of the stack tops using 4 64-bit
bitsets:

```
	White    uint64
	Black    uint64
	Standing uint64
	Caps     uint64
```

`White` and `Black` store the color of the topmost piece in each
stack, and are mutually exclusive.

`Standing` and `Caps` store whether the topmost piece is a standing
stone or capstone, respectively, and are also exclusive. A piece
present in `White` or `Black` but not in `Standing` or `Caps` is a
road.

This representation affords very efficient calculation of several
valuable board features:

- `White&Caps` gives the location of white's capstone(s).
- `White&^Standing` gives a bitset containing all positions that may
  be part of a road for White.
- `popcount(White&^(Standing|Caps))` gives white's current flat count.

(and vice-versa for black).

# Stacks

We begin by noting that pieces in a stack are constrained to be flats,
and therefore a single bit suffices to represent a single piece in a
stack. By convention, we'll assign `1` to black, and `0` to white.

We can then represent a single stack by defining its height, and
defining its pieces as a set of bits.

For 6x6 and smaller, a `uint64` suffices to represent the highest
possible stack, even assuming all available pieces were to be stacked
atop each other.

(efficiently handling the rare but hypothetically possible overflow
case on 8x8 is an open problem).

We therefore define the stacks by using two parallel arrays, one
holding height, and one holding the stack contents:


```
	Height []uint8
    Stacks []uint64
```

By convention, in Taktician, `Height` stores the height *including*
the top layer of the stack, but `Stacks` omits the top piece in the
stack. Thus, the low `Height[i]-1` bits of `Stacks[i]` are
significant. Taktician uses the lsb to represent the top of the stack.

Stacking and unstacking are fairly simply implemented by bit shifts
and masks. Managing the interplay between the board top and the stacks
requires some finesse, but was deemed worth it in light of the
significance of the board top. Further experimentation and
benchmarking may be in order, however.
