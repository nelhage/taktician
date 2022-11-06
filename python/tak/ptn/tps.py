import tak


def parse_tps(tps):
    bits = tps.split(" ")
    if len(bits) != 3:
        raise IllegalTPS("need three components")
    board, who, move = bits

    if not who in "12":
        raise IllegalTPS("Current player must be either 1 or 2")
    try:
        ply = 2 * (int(move) - 1) + int(who) - 1
    except ValueError:
        raise IllegalTPS("Bad move number: " + move)

    squares = []
    rows = board.split("/")
    for row in reversed(rows):
        rsq = parse_row(row)
        if len(rsq) != len(rows):
            raise IllegalTPS("inconsistent size")
        squares += rsq

    return tak.Position.from_squares(tak.Config(size=len(rows)), squares, ply)


def parse_row(rtext):
    squares = []
    bits = rtext.split(",")
    for b in bits:
        if b[0] == "x":
            n = 1
            if len(b) > 1:
                n = int(b[1:])
            squares += [[]] * n
            continue

        stack = []
        for c in b:
            if c == "1":
                stack.append(tak.Piece.cached(tak.Color.WHITE, tak.Kind.FLAT))
            elif c == "2":
                stack.append(tak.Piece.cached(tak.Color.BLACK, tak.Kind.FLAT))
            elif c in ("C", "S"):
                if not stack:
                    raise IllegalTPS("bare capstone or standing")
                typ = tak.Kind.CAPSTONE if c == "C" else tak.Kind.STANDING
                stack[-1] = tak.Piece.cached(stack[-1].color, typ)
            else:
                raise IllegalTPS("bad character: " + c)

        squares.append(list(reversed(stack)))
    return squares


def format_tps(pos):
    rows = []
    for row in range(pos.size):
        i = row * pos.size
        rows.append(_format_row(pos.board[i : i + pos.size]))

    return " ".join(
        ["/".join(reversed(rows)), str((pos.ply % 2) + 1), str(pos.ply // 2 + 1)]
    )


def _format_row(row):
    out = []
    i = 0
    while i < len(row):
        x = 0
        while i + x < len(row) and row[i + x] == []:
            x += 1
        if x > 0:
            out.append("x{0}".format(x if x > 1 else ""))
            i += x
        else:
            out.append(_format_square(row[i]))
            i += 1
    return ",".join(out)


def _format_square(sq):
    out = []
    for p in reversed(sq):
        if p.color == tak.Color.WHITE:
            out.append("1")
        else:
            out.append("2")

    if sq[0].kind == tak.Kind.STANDING:
        out.append("S")
    elif sq[0].kind == tak.Kind.CAPSTONE:
        out.append("C")

    return "".join(out)


class IllegalTPS(Exception):
    pass


__all__ = ["parse_tps", "format_tps", "IllegalTPS"]
