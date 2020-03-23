package prove

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/nelhage/taktician/tak"
)

type Evaluation int8

const (
	EvalUnknown Evaluation = iota
	EvalTrue
	EvalFalse
)

const (
	flagIrreversible = 1 << iota
	flagExpanded     = 1 << iota
)

const inf = ^uint32(0)

func saturatingAdd(l uint32, r uint32) uint32 {
	if (l + r) < l {
		return inf
	}
	return l + r
}

type node struct {
	parent          *node
	position        *tak.Position
	proof, disproof uint32

	value Evaluation
	flags int32

	children []*node
}

func (n *node) expanded() bool {
	return n.flags&flagExpanded != 0
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
}

type Config struct {
	Debug int
}

type Prover struct {
	cfg    *Config
	stats  Stats
	player tak.Color
	root   *node
}

func New(cfg Config) *Prover {
	return &Prover{
		cfg: &cfg,
	}
}

type ProofResult struct {
	Duration time.Duration
	Result   Evaluation
	Stats    Stats
}

func (p *Prover) Prove(ctx context.Context, pos *tak.Position) ProofResult {
	p.player = pos.ToMove()
	start := time.Now()
	p.prove(ctx, pos)
	if p.root.proof == 0 {
		p.root.value = EvalTrue
	} else if p.root.disproof == 0 {
		p.root.value = EvalFalse
	}
	return ProofResult{
		Result:   p.root.value,
		Stats:    p.stats,
		Duration: time.Since(start),
	}
}

const kProgressFrequency = 10000
const kCheckDoneFrequency = 1000

func (p *Prover) prove(ctx context.Context, pos *tak.Position) {
	start := time.Now()
	p.stats.Nodes += 1
	p.root = &node{
		position: pos,
		parent:   nil,
	}
	p.evaluate(p.root)
	p.setNumbers(p.root)
	var i uint64
	current := p.root
Outer:
	for p.root.proof != 0 && p.root.disproof != 0 {
		i++
		next := p.selectMostProving(current)

		if i%kProgressFrequency == 0 && p.cfg.Debug > 0 {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			log.Printf("time=%s nodes=%d live=%d done=%d/%d/%d root=(%d, %d) heap=%d",
				time.Since(start),
				p.stats.Nodes,
				p.stats.Nodes-(p.stats.Proved+p.stats.Disproved+p.stats.Dropped),
				p.stats.Proved,
				p.stats.Disproved,
				p.stats.Dropped,
				p.root.proof,
				p.root.disproof,
				stats.HeapAlloc,
			)
			if p.cfg.Debug > 1 {
				log.Printf("  children=%s", formatChildren(p.root.children))
			}
		}
		if i%kCheckDoneFrequency == 0 {
			select {
			case <-ctx.Done():
				break Outer
			default:
			}
		}

		p.expand(next)
		current = p.updateAncestors(next)
	}
}

func (p *Prover) checkRepetition(n *node) bool {
	if (n.flags & flagIrreversible) != 0 {
		return false
	}
	count := 1
	walk := n.parent
	for walk != nil && (walk.flags&flagIrreversible) == 0 && count < 3 {
		if walk.position.Equal(n.position) {
			count += 1
		}
		walk = walk.parent
	}
	return count == 3
}

func (p *Prover) evaluate(node *node) {
	if over, who := node.position.GameOver(); over {
		if who == p.player {
			node.value = EvalTrue
		} else {
			node.value = EvalFalse
		}
	} else {
		if p.checkRepetition(node) {
			node.value = EvalFalse
		} else {
			node.value = EvalUnknown
		}

	}
}

func (p *Prover) setNumbers(node *node) {
	if node.expanded() {
		if p.andNode(node) {
			node.proof = 0
			node.disproof = inf
			for _, c := range node.children {
				node.proof = saturatingAdd(node.proof, c.proof)
				if c.disproof < node.disproof {
					node.disproof = c.disproof
				}
			}
		} else {
			node.proof = inf
			node.disproof = 0
			for _, c := range node.children {
				node.disproof = saturatingAdd(node.disproof, c.disproof)
				if c.proof < node.proof {
					node.proof = c.proof
				}
			}
		}
	} else {
		switch node.value {
		case EvalTrue:
			node.proof = 0
			node.disproof = inf
		case EvalFalse:
			node.proof = inf
			node.disproof = 0
		case EvalUnknown:
			node.proof = 1
			node.disproof = 1
		}
	}
}

func formatChildren(children []*node) string {
	var buf bytes.Buffer
	for _, c := range children {
		fmt.Fprintf(&buf, "(%d, %d) ", c.proof, c.disproof)
	}
	return buf.String()
}

func (p *Prover) selectMostProving(current *node) *node {
	for current.expanded() {
		var child *node
		if p.andNode(current) {
			for _, c := range current.children {
				if c.disproof == current.disproof {
					child = c
					break
				}
			}

		} else {
			for _, c := range current.children {
				if c.proof == current.proof {
					child = c
					break
				}
			}
		}
		if child == nil {
			var ty string
			if p.andNode(current) {
				ty = "AND"
			} else {
				ty = "OR"
			}
			log.Printf("consistency error depth=%d type=%s proof=%d disproof=%d",
				current.depth(),
				ty,
				current.proof,
				current.disproof,
			)
			log.Printf("children: %s", formatChildren(current.children))
			panic("consistency error")
		}
		current = child
	}
	return current
}

func (p *Prover) andNode(n *node) bool {
	return n.position.ToMove() != p.player
}

func (p *Prover) expand(n *node) {
	var buffer [30]tak.Move
	allMoves := n.position.AllMoves(buffer[:])
	for _, m := range allMoves {
		cn, e := n.position.Move(m)
		if e != nil {
			continue
		}
		p.stats.Nodes += 1
		child := &node{
			position: cn,
			parent:   n,
		}

		dx, dy := m.Dest()
		reversible := m.IsSlide() && n.position.Top(int(dx), int(dy)).Kind() != tak.Standing
		if !reversible {
			child.flags |= flagIrreversible
		}
		p.evaluate(child)
		p.setNumbers(child)
		n.children = append(n.children, child)
		if (p.andNode(n) && child.proof == 0) || (!p.andNode(n) && child.disproof == 0) {
			break
		}
	}
	n.flags |= flagExpanded
}

func (p *Prover) updateAncestors(node *node) *node {
	for true {
		oldproof := node.proof
		olddisproof := node.disproof
		p.setNumbers(node)
		if node.proof == 0 || node.disproof == 0 {
			if node.proof == 0 {
				p.stats.Proved += 1
				if !p.andNode(node) {
					p.stats.Dropped += uint64(len(node.children) - 1)
				}
			} else {
				p.stats.Disproved += 1
				if p.andNode(node) {
					p.stats.Dropped += uint64(len(node.children) - 1)
				}
			}
			node.children = nil
		} else if node.proof == oldproof && node.disproof == olddisproof {
			return node
		}

		if node.parent == nil {
			return node
		}
		node = node.parent
	}
	return node
}
