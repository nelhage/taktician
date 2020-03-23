package prove

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/nelhage/taktician/ptn"
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
	flagExpanded     = 1 << iota
)

const inf = ^uint64(0)

type node struct {
	parent          *node
	position        *tak.Position
	proof, disproof uint64
	move            tak.Move

	value evaluation
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

type prover struct {
	stats struct {
		nodes     uint64
		proved    uint64
		disproved uint64
		dropped   uint64
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

		if i%kProgressFrequency == 0 {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			log.Printf("time=%s nodes=%d live=%d done=%d/%d/%d root=(%d, %d) heap=%d",
				time.Since(start),
				p.stats.nodes,
				p.stats.nodes-(p.stats.proved+p.stats.disproved+p.stats.dropped),
				p.stats.proved,
				p.stats.disproved,
				p.stats.dropped,
				p.root.proof,
				p.root.disproof,
				stats.HeapAlloc,
			)
			/*
				log.Printf("  children=%s", formatChildren(p.root.children))
				log.Printf("  line=%s", formatLine(next))
			*/

		}

		p.expand(next)
		current = p.updateAncestors(next)
	}
	log.Printf("Done in %s, nodes=%d proof=%d disproof=%d",
		time.Since(start),
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
			if node.disproof == 0 {
				node.proof = inf
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
			if node.proof == 0 {
				node.disproof = inf
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

func formatChildren(children []*node) string {
	var buf bytes.Buffer
	for _, c := range children {
		fmt.Fprintf(&buf, "(%d, %d) ", c.proof, c.disproof)
	}
	return buf.String()
}

func formatLine(node *node) string {
	var bits []string
	for node != nil && node.parent != nil {
		bits = append(bits, fmt.Sprintf("%s@(%d, %d)",
			ptn.FormatMove(node.move), node.proof, node.disproof))
		node = node.parent
	}
	for i := 0; i < len(bits)/2; i += 1 {
		bits[i], bits[len(bits)-1-i] = bits[len(bits)-1-i], bits[i]
	}
	return strings.Join(bits, " ")
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
			log.Printf("children: %s", formatChildren(current.children))
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
		n.children = append(n.children, child)
		if (p.andNode(n) && child.proof == 0) || (!p.andNode(n) && child.disproof == 0) {
			break
		}
	}
	n.flags |= flagExpanded
}

func (p *prover) updateAncestors(node *node) *node {
	for true {
		oldproof := node.proof
		olddisproof := node.disproof
		p.setNumbers(node)
		if node.proof == 0 || node.disproof == 0 {
			if node.proof == 0 {
				p.stats.proved += 1
				if !p.andNode(node) {
					p.stats.dropped += uint64(len(node.children) - 1)
				}
			} else {
				p.stats.disproved += 1
				if p.andNode(node) {
					p.stats.dropped += uint64(len(node.children) - 1)
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
