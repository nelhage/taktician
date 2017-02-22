import tak

def parse_tps(tps):
  bits = tps.split(' ')
  if len(bits) != 3:
    raise IllegalTPS("need three components")
  board, who, move = bits

  if not who in '12':
    raise IllegalTPS("Current player must be either 1 or 2")
  try:
    ply = 2 * (int(move)-1) + int(who)-1
  except ValueError:
    raise IllegalTPS("Bad move number: " + move)

  squares = []
  rows = board.split("/")
  for row in rows:
    rsq = parse_row(row)
    if len(rsq) != len(rows):
      raise IllegalTPS("inconsistent size")
    squares += rsq

  return tak.Position.from_squares(
    tak.Config(size = len(rows)),
    squares,
    ply)

def parse_row(rtext):
  squares = []
  bits = rtext.split(',')
  for b in bits:
    if b[0] == 'x':
      n = 1
      if len(b) > 1:
        n = int(b[1:])
      squares += [[]] * n
      continue

    stack = []
    for c in b:
      if c == '1':
        stack.append(tak.Piece(tak.Color.WHITE, tak.Kind.FLAT))
      elif c == '2':
        stack.append(tak.Piece(tak.Color.BLACK, tak.Kind.FLAT))
      elif c in ('C', 'S'):
        if not stack:
          raise IllegalTPS("bare capstone or standing")
        typ = tak.Kind.CAPSTONE if c == 'C' else tak.Kind.STANDING
        stack[-1] = tak.Piece(stack[-1].color, typ)
      else:
        raise IllegalTPS("bad character: " + c)

    squares.append(reversed(stack))
  return squares

class IllegalTPS(Exception):
  pass


__all__ = ['parse_tps', 'IllegalTPS']
