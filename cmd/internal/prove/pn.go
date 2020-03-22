package prove

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/nelhage/taktician/tak"
)

type evaluation int8

const (
	evalUnknown evaluation = iota
	evalTrue
	evalFalse
)

const (
	flagIrreversible = 1 << iota
)

const inf = ^uint64(0)

type node struct {
	parent          *node
	position        *tak.Position
	proof, disproof uint64

	value evaluation
	flags int32

	children []*node
}

func (n *node) expanded() bool {
	return n.children != nil
}

func (n *node) depth() int {
	d := 0
	for n.parent != nil {
		n = n.parent
		d += 1
	}
	return d
}

type prover struct {
	stats struct {
		nodes     uint64
		proved    uint64
		disproved uint64
	}
	player tak.Color
	root   *node
}

func prove(pos *tak.Position) {
	p := prover{
		player: pos.ToMove(),
	}
	p.prove(pos)
}

const kProgressFrequency = 10000

func (p *prover) prove(pos *tak.Position) {
	start := time.Now()
	p.stats.nodes += 1
	p.root = &node{
		position: pos,
		parent:   nil,
	}
	p.evaluate(p.root)
	p.setNumbers(p.root)
	var i uint64
	current := p.root
	for p.root.proof != 0 && p.root.disproof != 0 {
		i++
		next := p.selectMostProving(current)
		p.expand(next)
		current = p.updateAncestors(next)
		if i%kProgressFrequency == 0 {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			log.Printf("time=%s nodes=%d proved=%d/%d root=(%d, %d) heap=%d",
				time.Now().Sub(start),
				p.stats.nodes,
				p.stats.proved,
				p.stats.disproved,
				p.root.proof,
				p.root.disproof,
				stats.HeapAlloc,
			)
		}
	}
	log.Printf("Done in %s, nodes=%d proof=%d disproof=%d",
		time.Now().Sub(start),
		p.stats.nodes,
		p.root.proof,
		p.root.disproof,
	)
}

func (p *prover) checkRepetition(n *node) bool {
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

func (p *prover) evaluate(node *node) {
	if over, who := node.position.GameOver(); over {
		if who == p.player {
			node.value = evalTrue
		} else {
			node.value = evalFalse
		}
	} else {
		if p.checkRepetition(node) {
			node.value = evalFalse
		} else {
			node.value = evalUnknown
		}

	}
}

func (p *prover) setNumbers(node *node) {
	if node.expanded() {
		if p.andNode(node) {
			node.proof = 0
			node.disproof = inf
			for _, c := range node.children {
				node.proof += c.proof
				if c.disproof < node.disproof {
					node.disproof = c.disproof
				}
			}
		} else {
			node.proof = inf
			node.disproof = 0
			for _, c := range node.children {
				node.disproof += c.disproof
				if c.proof < node.proof {
					node.proof = c.proof
				}
			}
		}
	} else {
		switch node.value {
		case evalTrue:
			node.proof = 0
			node.disproof = inf
		case evalFalse:
			node.proof = inf
			node.disproof = 0
		case evalUnknown:
			node.proof = 1
			node.disproof = 1
		}
	}
}

func (p *prover) selectMostProving(current *node) *node {
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
			var buf bytes.Buffer
			for _, c := range current.children {
				fmt.Fprintf(&buf, "(%d, %d) ", c.proof, c.disproof)
			}
			log.Printf("children: %s", buf.String())
			panic("consistency error")
		}
		current = child
	}
	return current
}

func (p *prover) andNode(n *node) bool {
	return n.position.ToMove() != p.player
}

func (p *prover) expand(n *node) {
	var buffer [30]tak.Move
	allMoves := n.position.AllMoves(buffer[:])
	for _, m := range allMoves {
		cn, e := n.position.Move(m)
		if e != nil {
			continue
		}
		p.stats.nodes += 1
		child := &node{
			position: cn,
			parent:   n,
			move:     m,
		}

		dx, dy := m.Dest()
		reversible := m.IsSlide() && n.position.Top(int(dx), int(dy)).Kind() != tak.Standing
		if !reversible {
			child.flags |= flagIrreversible
		}
		p.evaluate(child)
		p.setNumbers(child)
		if (p.andNode(n) && child.proof == 0) || (!p.andNode(n) && child.disproof == 0) {
			break
		}
		n.children = append(n.children, child)
	}
}

func (p *prover) updateAncestors(node *node) *node {
	for true {
		oldproof := node.proof
		olddisproof := node.disproof
		p.setNumbers(node)
		if node.proof == oldproof && node.disproof == olddisproof {
			return node
		}
		if node.proof == 0 || node.disproof == 0 {
			node.children = nil
			if node.proof == 0 {
				p.stats.proved += 1
			} else {
				p.stats.disproved += 1
			}
		}
		if node.parent == nil {
			return node
		}
		node = node.parent
	}
	return node
}
