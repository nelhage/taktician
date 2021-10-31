package prove

import (
	"log"
	"time"
	"unsafe"

	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

type DFPNSolver struct {
	attacker tak.Color
	table    dfpnTable
	debug    int

	stack []dfpnFrame
}

type dfpnFrame struct {
	g *tak.Position
	m tak.Move
}

type DFPNConfig struct {
	Debug    int
	Attacker tak.Color
	TableMem int64
}

type proofNumbers struct {
	phi, delta uint32
}

const INFINITY uint32 = 1 << 30

func (pn proofNumbers) exceeded(other proofNumbers) bool {
	return pn.phi >= other.phi || pn.delta >= other.delta
}

func (pn *proofNumbers) solved() bool {
	return pn.phi == 0 || pn.delta == 0
}

type dfpnTable struct {
	entries []entry
}

func (t *dfpnTable) lookup(g *tak.Position) (entry, bool) {
	e := t.entries[g.Hash()%uint64(len(t.entries))]
	if e.hash == g.Hash() {
		return e, true
	}
	return entry{}, false
}

func (t *dfpnTable) store(e *entry) bool {
	target := &t.entries[e.hash%uint64(len(t.entries))]
	if target.work <= e.work {
		*target = *e
		return true
	}

	return false
}

type entry struct {
	bounds proofNumbers
	hash   uint64
	work   uint64
	pv     tak.Move
	//	child  uint8
}

type dfpnChild struct {
	data entry
	move tak.Move
	g    *tak.Position
}

const defaultTableMem = 100 * 1024 * 1024

func NewDFPN(cfg *DFPNConfig) *DFPNSolver {
	if cfg.TableMem <= 0 {
		cfg.TableMem = defaultTableMem
	}
	return &DFPNSolver{
		attacker: cfg.Attacker,
		table: dfpnTable{
			entries: make([]entry, cfg.TableMem/int64(unsafe.Sizeof(entry{}))),
		},
		debug: cfg.Debug,
	}
}

func (d *DFPNSolver) Prove(g *tak.Position) ProofResult {
	if d.attacker == tak.NoColor {
		d.attacker = g.ToMove()
	}
	d.stack = nil
	start := time.Now()
	entry, _ := d.mid(g, proofNumbers{phi: INFINITY / 2, delta: INFINITY / 2}, entry{
		hash:   g.Hash(),
		work:   0,
		bounds: proofNumbers{phi: 1, delta: 1},
	})
	duration := time.Since(start)
	var result Evaluation = EvalUnknown
	if entry.bounds.phi == 0 {
		result = EvalTrue
	} else if entry.bounds.delta == 0 {
		result = EvalFalse
	}
	return ProofResult{
		Result:   result,
		Move:     entry.pv,
		Proof:    entry.bounds.phi,
		Disproof: entry.bounds.delta,
		Duration: duration,
	}
}

func (d *DFPNSolver) checkRepetition() bool {
	if len(d.stack) == 0 {
		return false
	}
	idx := len(d.stack) - 1
	cur := d.stack[idx].g.Hash()
	count := 0

	for idx > 0 {
		if d.stack[idx].g.Hash() == cur {
			count++
		}
		if count == 3 {
			return true
		}
		switch d.stack[idx].m.Type {
		case tak.PlaceFlat:
		case tak.PlaceStanding:
		case tak.PlaceCapstone:
			break
		}
		idx--
	}
	return false
}

func (d *DFPNSolver) mid(g *tak.Position, bounds proofNumbers, current entry) (entry, uint64) {
	if current.bounds.exceeded(bounds) {
		return current, 0
	}

	if over, result := g.GameOver(); over {
		current.bounds = d.terminalBounds(g, result)
		return current, 0
	}
	if d.checkRepetition() {
		current.bounds = d.terminalBounds(g, tak.NoColor)
		return current, 0
	}

	if d.debug > 6 && len(d.stack) > 0 {
		log.Printf(" depth=%d toMove=%s move=%s current=(%d,%d) bounds=(%d,%d)",
			len(d.stack),
			g.ToMove(),
			ptn.FormatMove(d.stack[len(d.stack)-1].m),
			current.bounds.phi, current.bounds.delta,
			bounds.phi, bounds.delta)
	}

	localWork := uint64(1)
	// compute children
	var children []dfpnChild
	var alloc [100]tak.Move
	moves := g.AllMoves(alloc[:])
	for _, m := range moves {
		p, err := g.Move(m)
		if err != nil {
			continue
		}
		childEntry := entry{
			hash:   p.Hash(),
			bounds: proofNumbers{1, 1},
		}
		if over, result := p.GameOver(); over {
			childEntry.bounds = d.terminalBounds(p, result)
		} else if b, ok := d.table.lookup(p); ok {
			childEntry = b
		}
		// todo look up table
		children = append(children, dfpnChild{
			move: m,
			g:    p,
			data: childEntry,
		})
		if childEntry.bounds.delta == 0 {
			break
		}
	}

	for {
		current.bounds = computePNs(children)
		if current.bounds.exceeded(bounds) {
			break
		}

		best_idx, childBounds := d.selectChild(children, bounds, current.bounds)
		// todo recurse

		current.pv = children[best_idx].move

		d.stack = append(d.stack, dfpnFrame{children[best_idx].g, children[best_idx].move})
		newEntry, work := d.mid(children[best_idx].g, childBounds, children[best_idx].data)
		d.stack = d.stack[:len(d.stack)-1]

		children[best_idx].data = newEntry
		localWork += work
		current.work += work
	}

	d.table.store(&current)

	return current, uint64(localWork)
}

func (d *DFPNSolver) terminalBounds(g *tak.Position, result tak.Color) proofNumbers {
	switch result {
	case tak.NoColor:
		if g.ToMove() == d.attacker {
			return proofNumbers{phi: INFINITY, delta: 0}
		} else {
			return proofNumbers{phi: 0, delta: INFINITY}
		}
	case g.ToMove():
		return proofNumbers{phi: 0, delta: INFINITY}
	default:
		return proofNumbers{phi: INFINITY, delta: 0}
	}
}

const epsilon = 0.1

func max(l, r uint32) uint32 {
	if l > r {
		return l
	}
	return r
}

func min(l, r uint32) uint32 {
	if l < r {
		return l
	}
	return r
}

func (d *DFPNSolver) selectChild(children []dfpnChild,
	bounds proofNumbers,
	pns proofNumbers) (int, proofNumbers) {
	delta1, delta2 := INFINITY, INFINITY

	best := -1
	for i, ch := range children {
		if ch.data.bounds.delta < delta1 {
			best = i
			delta2 = delta1
			delta1 = ch.data.bounds.delta
		} else if ch.data.bounds.delta < delta2 {
			delta2 = ch.data.bounds.delta
		}
	}

	phi1 := children[best].data.bounds.phi

	return best, proofNumbers{
		phi: bounds.delta + phi1 - pns.delta,
		delta: min(
			bounds.phi,
			//			max(delta2+1, uint32(float64(delta2)*(1.0+epsilon))),
			delta2+1,
		),
	}
}

func computePNs(children []dfpnChild) proofNumbers {
	out := proofNumbers{phi: INFINITY, delta: 0}
	for _, ch := range children {
		out.delta += ch.data.bounds.phi
		if out.delta > INFINITY {
			out.delta = INFINITY
		}
		if ch.data.bounds.delta < out.phi {
			out.phi = ch.data.bounds.delta
		}
	}
	return out
}
