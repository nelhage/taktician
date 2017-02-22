import tak

import attr

import re

@attr.s
class PTN(object):
  tags  = attr.ib()
  moves = attr.ib()

  @classmethod
  def parse(cls, text):
    head, tail = text.split("\n\n", 1)
    tags_ = re.findall(r'^\[(\w+) "([^"]+)"\]$', head, re.M)
    tags = dict(tags_)

    tail = re.sub(r'{[^}]+', ' ', tail)

    moves = []
    tokens = re.split(r'\s+', tail)
    for t in tokens:
      if t == '--':
        continue
      if re.search(r'\A(0|R|F|1|1/2)-(0|R|F|1|1/2)\Z', t):
        continue
      if re.match(r'\A\d+\.\Z', r):
        continue

      m = parse_move(t)
      moves.append(m)
    return cls(tags = tags, moves = moves)

slide_map = {
  '-': tak.MoveType.SLIDE_DOWN,
  '+': tak.MoveType.SLIDE_UP,
  '<': tak.MoveType.SLIDE_LEFT,
  '>': tak.MoveType.SLIDE_RIGHT,
}
slide_rmap = dict((v, k) for (k, v) in slide_map.items())

place_map = {
  '':  tak.MoveType.PLACE_FLAT,
  'S': tak.MoveType.PLACE_STANDING,
  'C': tak.MoveType.PLACE_CAPSTONE,
  'F': tak.MoveType.PLACE_FLAT,
}
place_rmap = {
  tak.MoveType.PLACE_FLAT: '',
  tak.MoveType.PLACE_STANDING: 'S',
  tak.MoveType.PLACE_CAPSTONE: 'C',
}

def parse_move(move):
  m = re.search(r'\A([CFS]?)([1-8]?)([a-h])([1-8])([<>+-]?)([1-8]*)[CFS]?\Z', move)
  if not m:
    raise BadMove(move, "malformed move")
  stone, pickup, file, rank, dir, drops = m.groups()

  x = ord(file) - ord('a')
  y = ord(rank) - ord('1')

  if pickup and not dir:
    raise BadMove(move, "pick up but no direction")

  typ = None
  if dir:
    typ = slide_map[dir]
  else:
    typ = place_map[stone]

  slides = None
  if drops:
    slides = [ord(c) - ord('0') for c in drops]

  if (drops or pickup) and not dir:
    raise BadMove(move, "pickup/drop without a direction")

  if dir and not pickup and not slides:
    pickup = '1'

  if pickup and not slides:
    slides = [int(pickup)]

  if pickup and int(pickup) != sum(slides):
    raise BadMove(move, "inconsistent pickup and drop: {0} v {1}".format(pickup, drops))

  return tak.Move(x, y, typ, slides)

def format_move(move):
  bits = []

  bits.append(place_rmap.get(move.type, ''))

  if move.type.is_slide():
    pickup = sum(move.slides)
    if pickup != 1:
      bits.append(pickup)

  bits.append(chr(move.x + ord('a')))
  bits.append(chr(move.y + ord('1')))

  if move.type.is_slide():
    bits.append(slide_rmap[move.type])

    if len(move.slides) > 1:
      bits += [chr(d + ord('0')) for d in move.slides]

  return ''.join(map(str, bits))

class BadMove(Exception):
  def __init__(self, text, error):
    self.move = text
    super().__init__(error)
