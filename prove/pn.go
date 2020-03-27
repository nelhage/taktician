package prove

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"runtime"
	"strconv"
	"time"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type Evaluation int8

const (
	EvalUnknown Evaluation = iota
	EvalTrue
	EvalFalse
)

func (e Evaluation) String() string {
	switch e {
	case EvalUnknown:
		return "unknown"
	case EvalTrue:
		return "proven"
	case EvalFalse:
		return "disproven"
	default:
		return fmt.Sprintf("ERR(%d)", e)
	}
}

const (
	flagIrreversible = 1 << iota
	flagExpanded
	flagAnd
)

const (
	inf = ^uint32(0)

	kCheckFrequency = 1000

	pn2Threshold = 1000
)

func saturatingAdd(l uint32, r uint32) uint32 {
	if (l + r) < l {
		return inf
	}
	return l + r
}

type node struct {
	parent *node
	move   tak.Move
	// phi -> proof for OR, disproof for AND
	// delta -> proof for AND, disproof for OR
	phi, delta uint32

	value      Evaluation
	flags      int8
	proofDepth uint16

	children []*node
}

func (n *node) proof() uint32 {
	if n.andNode() {
		return n.delta
	} else {
		return n.phi
	}
}

func (n *node) disproof() uint32 {
	if n.andNode() {
		return n.phi
	} else {
		return n.delta
	}
}

func (n *node) expanded() bool {
	return n.flags&flagExpanded != 0
}

func (n *node) andNode() bool {
	return (n.flags & flagAnd) != 0
}

func (n *node) depth() int {
	d := 0
	for n.parent != nil {
		n = n.parent
		d += 1
	}
	return d
}

type Stats struct {
	Nodes     uint64
	Proved    uint64
	Disproved uint64
	Dropped   uint64
	Expanded  uint64
	MaxDepth  uint64
}

func (st *Stats) Live() uint64 {
	return st.Nodes - (st.Proved + st.Disproved + st.Dropped)
}

type Config struct {
	Debug          int
	MaxNodes       uint64
	LogPrefix      string
	PreserveSolved bool
	PN2            bool
}

type Prover struct {
	ctx context.Context

	cfg   *Config
	stats Stats

	start time.Time

	root     *node
	position *tak.Position

	checkNode *node
	stack     []*tak.Position
	alloc     []*tak.Position

	moveBuffer [100]tak.Move

	progress <-chan time.Time
}

func New(cfg Config) *Prover {
	return &Prover{
		cfg: &cfg,
	}
}

type ProofResult struct {
	Duration        time.Duration
	Result          Evaluation
	Depth           uint32
	Stats           Stats
	Proof, Disproof uint32
	Move            tak.Move
}

func (p *Prover) Prove(ctx context.Context, pos *tak.Position) ProofResult {
	p.start = time.Now()
	p.stats = Stats{}
	p.position = pos
	p.ctx = ctx

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	p.progress = ticker.C

	p.prove(ctx, pos)

	var pv tak.Move
	if p.root.phi == 0 {
		p.root.value = EvalTrue
		for _, c := range p.root.children {
			if c.delta == 0 {
				pv = c.move
			}
		}
	} else if p.root.delta == 0 {
		p.root.value = EvalFalse
		var best *node
		for _, c := range p.root.children {
			if best == nil || best.proofDepth < c.proofDepth {
				best = c
			}
		}
		if best != nil {
			pv = best.move
		}

	}

	return ProofResult{
		Result:   p.root.value,
		Stats:    p.stats,
		Duration: time.Since(p.start),
		Proof:    p.root.proof(),
		Disproof: p.root.disproof(),
		Move:     pv,
		Depth:    uint32(p.root.proofDepth),
	}
}

func name(n string) xml.Name {
	return xml.Name{Space: "", Local: n}
}

func elt(e *xml.Encoder, el xml.StartElement, inner func(*xml.Encoder)) {
	e.EncodeToken(el)
	inner(e)
	e.EncodeToken(xml.EndElement{Name: el.Name})
}

func (p *Prover) DumpTree(out io.Writer) {
	e := xml.NewEncoder(out)
	elt(e, xml.StartElement{Name: name("PNTree")},
		func(e *xml.Encoder) {
			p.walkTree(e, p.root)
		})
	if err := e.Flush(); err != nil {
		panic(fmt.Sprintf("flush: %v", err))
	}
}

func (p *Prover) walkTree(e *xml.Encoder, node *node) {
	var ty string
	if node.andNode() {
		ty = "AND"
	} else {
		ty = "OR"
	}
	elt(e, xml.StartElement{Name: name("Node"),
		Attr: []xml.Attr{
			{Name: name("Move"), Value: ptn.FormatMove(node.move)},
			{Name: name("Type"), Value: ty},
			{Name: name("Phi"), Value: strconv.FormatUint(uint64(node.phi), 10)},
			{Name: name("Delta"), Value: strconv.FormatUint(uint64(node.delta), 10)},
			{Name: name("Depth"), Value: strconv.FormatUint(uint64(node.proofDepth), 10)},
			{Name: name("Value"), Value: node.value.String()},
		},
	}, func(e *xml.Encoder) {
		if node.expanded() {
			elt(e, xml.StartElement{Name: name("Children")},
				func(e *xml.Encoder) {
					for _, c := range node.children {
						p.walkTree(e, c)
					}
				})
		}
	})
}

func (p *Prover) prove(ctx context.Context, pos *tak.Position) {
	p.stats.Nodes += 1
	p.root = &node{
		parent: nil,
	}
	p.alloc = []*tak.Position{pos}
	p.stack = p.alloc
	p.checkNode = p.root
	p.evaluate(p.root)
	p.setNumbers(p.root)
	p.search(ctx, p.cfg.MaxNodes)
}

func (p *Prover) search(ctx context.Context, maxNodes uint64) {
	var i uint64
	current := p.root
Outer:
	for p.root.phi != 0 && p.root.delta != 0 {
		i++
		next := p.selectMostProving(current)

		if i%kCheckFrequency == 0 || p.cfg.PN2 {
			select {
			case <-ctx.Done():
				break Outer
			default:
			}
			select {
			case <-p.progress:
				var stats runtime.MemStats
				runtime.ReadMemStats(&stats)
				log.Printf("%stime=%s nodes=%d live=%d done=%d/%d/%d expanded=%d root=(%d, %d) heap=%s",
					p.cfg.LogPrefix,
					time.Since(p.start),
					p.stats.Nodes,
					p.stats.Live(),
					p.stats.Proved,
					p.stats.Disproved,
					p.stats.Dropped,
					p.stats.Expanded,
					p.root.phi,
					p.root.delta,
					formatBytes(stats.HeapAlloc),
				)
				if p.cfg.Debug > 1 {
					log.Printf("%s  children=%s", p.cfg.LogPrefix, formatChildren(p.root.children))
				}
			default:
			}
		}
		if maxNodes > 0 && p.stats.Live() > maxNodes {
			break Outer
		}

		p.expand(next)
		current = p.updateAncestors(next)
	}
	for p.checkNode != p.root {
		p.ascend()
	}
}

func (p *Prover) checkRepetition(n *node) bool {
	if (n.flags & flagIrreversible) != 0 {
		return false
	}
	count := 1
	current := p.currentPosition(n)
	walk := n.parent
	i := len(p.stack) - 2
	for walk != nil && (walk.flags&flagIrreversible) == 0 && count < 3 {
		if p.stack[i].Equal(current) {
			count += 1
		}
		walk = walk.parent
		i -= 1
	}
	return count == 3
}

func (p *Prover) evaluate(node *node) {
	if over, who := p.currentPosition(node).GameOver(); over {
		if who == p.position.ToMove() {
			node.value = EvalTrue
		} else {
			node.value = EvalFalse
		}
	} else if p.checkRepetition(node) {
		node.value = EvalFalse
	} else {
		node.value = EvalUnknown
	}
}

func (p *Prover) setNumbers(node *node) {
	if node.expanded() {
		node.delta = 0
		node.phi = inf
		for _, c := range node.children {
			node.delta = saturatingAdd(node.delta, c.phi)
			if c.delta < node.phi {
				node.phi = c.delta
			}
		}
	} else {
		switch node.value {
		case EvalTrue, EvalFalse:
			if node.andNode() == (node.value == EvalTrue) {
				node.phi = inf
				node.delta = 0
			} else {
				node.phi = 0
				node.delta = inf
			}
		case EvalUnknown:
			pos := p.currentPosition(node)
			stones := uint32(pos.BlackStones() + pos.WhiteStones())
			node.phi = 1
			node.delta = stones
		}
	}
}

func formatChildren(children []*node) string {
	var buf bytes.Buffer
	for _, c := range children {
		fmt.Fprintf(&buf, "(%d, %d) ", c.phi, c.delta)
	}
	return buf.String()
}

var sizeTables = []struct {
	order  int
	suffix string
}{
	{40, "T"},
	{30, "G"},
	{20, "M"},
	{10, "K"},
	{0, "B"},
}

func formatBytes(bytes uint64) string {
	for _, e := range sizeTables {
		if bytes > 10*(1<<e.order) {
			return fmt.Sprintf("%d%s", bytes>>e.order, e.suffix)
		}
	}
	return fmt.Sprintf("%dB", bytes)
}

func (p *Prover) selectMostProving(current *node) *node {
	for current.expanded() {
		var child *node
		for _, c := range current.children {
			if c.delta == current.phi {
				child = c
				break
			}
		}
		if child == nil {
			var ty string
			if current.andNode() {
				ty = "AND"
			} else {
				ty = "OR"
			}
			log.Printf("consistency error depth=%d type=%s delta=%d phi=%d",
				current.depth(),
				ty,
				current.phi,
				current.delta,
			)
			log.Printf("children: %s", formatChildren(current.children))
			panic("consistency error")
		}
		if !p.tryDescend(child) {
			panic("failed to descend")
		}
		current = child
	}
	return current
}

func (p *Prover) pn2(n *node) {
	oldRoot := p.root
	oldStats := p.stats

	p.root = n
	p.stats = Stats{}
	p.cfg.PN2 = false
	p.cfg.LogPrefix = " [PN₂]"

	p.search(p.ctx, oldStats.Live())
	if n.proof() == 0 {
		n.value = EvalTrue
	} else if n.disproof() == 0 {
		n.value = EvalFalse
	}

	if p.cfg.Debug > 2 {
		log.Printf("[pn2] depth=%d(%d) val=%s limit=%d searched=%d pn=(%d,%d)",
			p.depth(),
			oldStats.MaxDepth,
			n.value,
			oldStats.Live(),
			p.stats.Nodes,
			n.phi,
			n.delta,
		)
	}

	if p.stats.MaxDepth > oldStats.MaxDepth {
		oldStats.MaxDepth = p.stats.MaxDepth
	}

	p.root = oldRoot
	p.stats = oldStats
	p.cfg.PN2 = true
	p.cfg.LogPrefix = ""

	for _, c := range n.children {
		c.flags &= ^flagExpanded
		c.children = nil
		p.stats.Nodes += 1
	}
	p.stats.Expanded += 1
}

func (p *Prover) expand(n *node) {
	current := p.currentPosition(n)

	if p.cfg.PN2 && p.stats.Nodes > pn2Threshold {
		p.pn2(n)
		return
	}

	allMoves := current.AllMoves(p.moveBuffer[:0])
	for _, m := range allMoves {
		child := &node{
			parent: n,
			move:   m,
		}

		if !p.tryDescend(child) {
			continue
		}
		p.stats.Nodes += 1

		dx, dy := m.Dest()
		reversible := m.IsSlide() && current.Top(int(dx), int(dy)).Kind() != tak.Standing
		if !reversible {
			child.flags |= flagIrreversible
		}
		if !n.andNode() {
			child.flags |= flagAnd
		}
		p.evaluate(child)
		p.setNumbers(child)
		p.ascend()
		n.children = append(n.children, child)
		if child.delta == 0 {
			break
		}
	}

	p.stats.Expanded += 1
	n.flags |= flagExpanded
	d := uint64(p.depth() + 1)
	if d > p.stats.MaxDepth {
		p.stats.MaxDepth = d
	}
}

func (p *Prover) updateAncestors(node *node) *node {
	for true {
		oldphi := node.phi
		olddelta := node.delta
		p.setNumbers(node)
		if node.phi == 0 || node.delta == 0 {
			if node.delta == 0 {
				var d uint16
				for _, c := range node.children {
					if c.phi != 0 {
						panic("inconsistent")
					}
					if c.proofDepth > d {
						d = c.proofDepth
					}
				}
				node.proofDepth = d + 1
			} else {
				d := uint16(1 << 15)
				for _, c := range node.children {
					if c.delta != 0 {
						continue
					}
					if c.proofDepth < d {
						d = c.proofDepth
					}
				}
				node.proofDepth = d + 1
			}
			if node.proof() == 0 {
				p.stats.Proved += 1
			} else {
				p.stats.Disproved += 1
			}
			if node.phi == 0 {
				p.stats.Dropped += uint64(len(node.children) - 1)
			}
			if node != p.root && !p.cfg.PreserveSolved {
				node.children = nil
			}
		} else if node.phi == oldphi && node.delta == olddelta {
			return node
		}

		if node == p.root {
			return node
		}
		node = node.parent
		p.ascend()
	}
	return node
}

func (p *Prover) tryDescend(n *node) bool {
	current := p.currentPosition(n.parent)
	var out *tak.Position
	if len(p.alloc) <= len(p.stack) {
		p.alloc = append(p.alloc, tak.Alloc(p.position.Size()))
	}
	out = p.alloc[len(p.stack)]
	_, err := current.MovePreallocated(n.move, out)
	if err != nil {
		return false
	}
	p.stack = p.alloc[0 : len(p.stack)+1]
	p.checkNode = n
	return true
}

func (p *Prover) currentPosition(cur *node) *tak.Position {
	if cur != p.checkNode {
		panic("inconsistent current position")
	}
	return p.stack[len(p.stack)-1]
}

func (p *Prover) depth() int {
	return len(p.stack) - 1
}

func (p *Prover) ascend() {
	p.stack = p.stack[0 : len(p.stack)-1]
	p.checkNode = p.checkNode.parent
}
