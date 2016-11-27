package tak

// Slides is essentially a packed [8]uint4, used to represent the
// slide counts in a Tak move in a space-efficient way. We store the
// first drop count in (s&0xf), the next in (s&0xf0), and so on.
type Slides uint32

func MkSlides(drops ...int) Slides {
	var out Slides
	for i := len(drops) - 1; i >= 0; i-- {
		if drops[i] > 8 {
			panic("bad drop")
		}
		out = out.Prepend(drops[i])
	}
	return out
}

func (s Slides) Len() int {
	l := 0
	for s != 0 {
		l++
		s >>= 4
	}
	return l
}

func (s Slides) Empty() bool {
	return s == 0
}

func (s Slides) Singleton() bool {
	return s > 0xf
}

func (s Slides) First() int {
	return int(s & 0xf)
}

func (s Slides) Prepend(next int) Slides {
	return (s << 4) | Slides(next)
}

type SlideIterator uint32

func (s Slides) Iterator() SlideIterator {
	return SlideIterator(s)
}

func (s SlideIterator) Next() SlideIterator {
	return s >> 4
}

func (s SlideIterator) Ok() bool {
	return s != 0
}

func (s SlideIterator) Elem() int {
	return int(s & 0xf)
}
