#!/usr/bin/env python
from .. import game, pieces, moves

import torch

MAX_RESERVES = 50
MAX_CAPSTONES = 2


class Token:
    EMPTY = 0

    WHITE_TOP_FLAT = 1
    WHITE_FLAT = 2
    WHITE_STANDING = 3
    WHITE_CAPSTONE = 4

    BLACK_TOP_FLAT = 5
    BLACK_FLAT = 6
    BLACK_STANDING = 7
    BLACK_CAPSTONE = 8

    WHITE_TO_PLAY = 9
    BLACK_TO_PLAY = 10

    LAST_CAPSTONE_VALUE = 255
    CAPSTONES = list(
        range(LAST_CAPSTONE_VALUE - MAX_CAPSTONES + 1, LAST_CAPSTONE_VALUE + 1)
    )
    FIRST_CAPSTONES_VALUE = CAPSTONES[0]
    LAST_RESERVES_VALUE = FIRST_CAPSTONES_VALUE - 1
    RESERVES = list(
        range(LAST_RESERVES_VALUE - MAX_RESERVES + 1, LAST_RESERVES_VALUE + 1)
    )

    FIRST_RESERVES_VALUE = RESERVES[0]

    # [to_play
    #   white_reserves white_caps
    #   black_reserves black_caps
    #   board...
    # ]


TOP_PIECES = {
    pieces.Piece(pieces.Color.WHITE, pieces.Kind.FLAT): Token.WHITE_TOP_FLAT,
    pieces.Piece(pieces.Color.BLACK, pieces.Kind.FLAT): Token.BLACK_TOP_FLAT,
    pieces.Piece(pieces.Color.WHITE, pieces.Kind.STANDING): Token.WHITE_STANDING,
    pieces.Piece(pieces.Color.BLACK, pieces.Kind.STANDING): Token.BLACK_STANDING,
    pieces.Piece(pieces.Color.WHITE, pieces.Kind.CAPSTONE): Token.WHITE_CAPSTONE,
    pieces.Piece(pieces.Color.BLACK, pieces.Kind.CAPSTONE): Token.BLACK_CAPSTONE,
}


def encode(p: game.Position) -> list[int]:
    data = []
    if p.to_move() == pieces.Color.WHITE:
        data.append(Token.WHITE_TO_PLAY)
    else:
        data.append(Token.BLACK_TO_PLAY)
    stones = p.stones
    data.append(Token.RESERVES[stones[0].stones])
    data.append(Token.CAPSTONES[stones[0].caps])
    data.append(Token.RESERVES[stones[1].stones])
    data.append(Token.CAPSTONES[stones[1].caps])

    for square in p.board:
        if len(square) == 0:
            data.append(Token.EMPTY)
            continue
        top, *stack = square
        data.append(TOP_PIECES[top])
        for flat in stack:
            data.append(
                Token.WHITE_FLAT
                if flat.color == pieces.Color.WHITE
                else Token.BLACK_FLAT
            )
    return data


def _encode_batch(
    inputs, encode_one, dtype=torch.float
) -> (torch.Tensor, torch.Tensor):
    lens = torch.empty((len(inputs),), dtype=torch.int)
    out = torch.zeros((len(inputs), 0), dtype=dtype)
    for (i, p) in enumerate(inputs):
        encoded = encode_one(p)
        if len(encoded) > out.size(1):
            tmp = torch.zeros((out.size(0), len(encoded)), dtype=out.dtype)
            tmp[:, : out.size(1)] = out
            out = tmp
        out[i, : len(encoded)] = torch.tensor(encoded, dtype=out.dtype)
        lens[i] = len(encoded)
    mask = torch.zeros_like(out, dtype=torch.bool)
    for i, l in enumerate(lens):
        mask[i, :l] = 1
    return out, mask


def encode_batch(positions) -> (torch.Tensor, torch.Tensor):
    return _encode_batch(positions, encode, dtype=torch.uint8)


def encode_move(size: int, m: moves.Move) -> list[int]:
    data = []
    data.append(size * m.y + m.x)
    data.append(m.type.value)
    if m.type.is_slide():
        data.append(moves.ALL_SLIDES[size].index(m.slides))
    return data


def encode_moves_batch(size, moves) -> (torch.Tensor, torch.Tensor):
    return _encode_batch(moves, lambda m: encode_move(size, m), dtype=torch.long)
