package prove

import (
	"log"
	"time"
	"unsafe"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/bitboard"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/tak"
)

const poolSize = 1024

type positionPool struct {
	buf  [poolSize]*tak.Position
	r, w int
}

type DFPNStats struct {
	Work       uint64
	Repetition uint64

	Terminal uint64
	Solved   uint64
	Hits     uint64
	Miss     uint64
}

type DFPNSolver struct {
	attacker tak.Color
	table    dfpnTable
	debug    int

	stats DFPNStats

	stack   []dfpnFrame
	killers []tak.Move

	pool positionPool

	c bitboard.Constants
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

func (d *DFPNSolver) Prove(g *tak.Position) (ProofResult, DFPNStats) {
	if d.attacker == tak.NoColor {
		d.attacker = g.ToMove()
	}
	d.c = bitboard.Precompute(uint(g.Size()))
	d.stats = DFPNStats{}

	d.stack = nil
	start := time.Now()
	entry, work := d.mid(g, proofNumbers{phi: INFINITY - 100, delta: INFINITY - 100}, entry{
		hash:   g.Hash(),
		work:   0,
		bounds: proofNumbers{phi: 1, delta: 1},
	})
	d.stats.Work = work
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
	}, d.stats
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

func (d *DFPNSolver) alloc() *tak.Position {
	if d.pool.r == d.pool.w {
		return nil
	}
	p := d.pool.buf[d.pool.r]
	d.pool.r += 1
	if d.pool.r == len(d.pool.buf) {
		d.pool.r = 0
	}
	return p
}

func (d *DFPNSolver) release(p *tak.Position) {
	if (d.pool.w+1)%len(d.pool.buf) == d.pool.r {
		return
	}
	d.pool.buf[d.pool.w] = p
	d.pool.w = (d.pool.w + 1) % len(d.pool.buf)
}

func (d *DFPNSolver) solve(p *tak.Position) (bool, tak.Color) {
	wp, wt, bp, bt := ai.CountThreats(&d.c, p)
	if wp+wt > 0 && p.ToMove() == tak.White {
		return true, tak.White
	}
	if bp+bt > 0 && p.ToMove() == tak.Black {
		return true, tak.Black
	}
	return false, tak.NoColor
}

func (d *DFPNSolver) mid(g *tak.Position, bounds proofNumbers, current entry) (entry, uint64) {
	if current.bounds.exceeded(bounds) {
		return current, 0
	}

	/*
		if over, result := g.GameOver(); over {
			current.bounds = d.terminalBounds(g, result)
			return current, 0
		}
	*/

	if d.checkRepetition() {
		d.stats.Repetition++
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
	var allocChildren [100]dfpnChild
	children := allocChildren[:0]
	var allocMoves [100]tak.Move
	moves := g.AllMoves(allocMoves[:0])
	defer func() {
		for _, ch := range children {
			d.release(ch.g)
		}
	}()

	depth := len(d.stack)
	var killer tak.Move
	if len(d.killers) > depth && d.killers[depth].Type != 0 {
		killer = d.killers[depth]
	}

	for _, m := range moves {
		alloc := d.alloc()
		p, err := g.MovePreallocated(m, alloc)
		if err != nil {
			d.release(alloc)
			continue
		}
		childEntry := entry{
			hash:   p.Hash(),
			bounds: proofNumbers{1, 1},
		}
		if over, result := p.GameOver(); over {
			d.stats.Terminal++
			childEntry.bounds = d.terminalBounds(p, result)
		} else if over, result := d.solve(p); over {
			d.stats.Solved++
			childEntry.bounds = d.terminalBounds(p, result)
		} else if b, ok := d.table.lookup(p); ok {
			d.stats.Hits++
			childEntry = b
		} else {
			d.stats.Miss++
			var buffer [100]tak.Move
			moves := len(p.AllMoves(buffer[:0]))
			childEntry.bounds = proofNumbers{phi: 1, delta: uint32(moves)}
		}
		children = append(children, dfpnChild{
			move: m,
			g:    p,
			data: childEntry,
		})
		if m == killer {
			children[0], children[len(children)-1] = children[len(children)-1], children[0]
		}

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

		current.pv = children[best_idx].move

		d.stack = append(d.stack, dfpnFrame{children[best_idx].g, children[best_idx].move})
		newEntry, work := d.mid(children[best_idx].g, childBounds, children[best_idx].data)
		d.stack = d.stack[:len(d.stack)-1]

		children[best_idx].data = newEntry
		localWork += work
		current.work += work
	}

	if current.bounds.phi == 0 {
		for len(d.killers) <= depth {
			d.killers = append(d.killers, tak.Move{})
		}
		d.killers[depth] = current.pv
	}

	d.table.store(&current)

	return current, uint64(localWork)
}

func (d *DFPNSolver) terminalBounds(g *tak.Position, result tak.Color) proofNumbers {
	if result == tak.NoColor {
		result = d.attacker.Flip()
	}
	if result == g.ToMove() {
		return proofNumbers{phi: 0, delta: INFINITY}
	} else {
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
			max(delta2+1, uint32(float64(delta2)*(1.0+epsilon))),
			// delta2+1,
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
