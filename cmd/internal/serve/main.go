package serve

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	"context"

	"google.golang.org/grpc"

	"github.com/google/subcommands"
	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/pb/tak/proto"
	"github.com/nelhage/taktician/ptn"
	"github.com/nelhage/taktician/symmetry"
	"github.com/nelhage/taktician/tak"
)

type Command struct {
	port int
}

func (*Command) Name() string     { return "serve" }
func (*Command) Synopsis() string { return "Seve Taktician RPCs via GRPC" }
func (*Command) Usage() string {
	return `serve
`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	flags.IntVar(&c.port, "port", 55430, "bind port")
}

type cache struct {
	sync.Mutex
	player  *ai.MinimaxAI
	cfg     ai.MinimaxConfig
	precise bool
}

type server struct {
	analyzeCache cache
	istakCache   cache
}

func (c *cache) getPlayer(size int, depth int, precise bool) *ai.MinimaxAI {
	if c.cfg.Size != size || c.cfg.Depth != int(depth) || c.precise != precise {
		c.cfg = ai.MinimaxConfig{
			Size:  size,
			Depth: depth,
			Debug: 1,
		}
		if precise {
			c.cfg.MakePrecise()
		}
		c.player = ai.NewMinimax(c.cfg)
		c.precise = precise
	}
	return c.player
}

func (s *server) Analyze(ctx context.Context, req *pb.AnalyzeRequest) (*pb.AnalyzeResponse, error) {
	p, e := ptn.ParseTPS(req.Position)
	if e != nil {
		return nil, e
	}

	s.analyzeCache.Lock()
	defer s.analyzeCache.Unlock()
	player := s.analyzeCache.getPlayer(p.Size(), int(req.Depth), req.Precise)

	var resp pb.AnalyzeResponse
	pv, value, _ := player.Analyze(ctx, p)
	for _, m := range pv {
		resp.Pv = append(resp.Pv, ptn.FormatMove(m))
	}
	resp.Value = value

	return &resp, nil
}

func (s *server) Canonicalize(ctx context.Context, req *pb.CanonicalizeRequest) (*pb.CanonicalizeResponse, error) {
	var ms []tak.Move
	for _, mstr := range req.Moves {
		mv, e := ptn.ParseMove(mstr)
		if e != nil {
			return nil, e
		}
		ms = append(ms, mv)
	}

	ms, e := symmetry.Canonical(int(req.Size), ms)
	if e != nil {
		return nil, e
	}

	var outms []string
	for _, m := range ms {
		outms = append(outms, ptn.FormatMove(m))
	}
	return &pb.CanonicalizeResponse{
		Moves: outms,
	}, nil
}

func (s *server) IsPositionInTak(ctx context.Context, req *pb.IsPositionInTakRequest) (*pb.IsPositionInTakResponse, error) {
	p, e := ptn.ParseTPS(req.Position)
	if e != nil {
		return nil, e
	}

	s.istakCache.Lock()
	defer s.istakCache.Unlock()
	player := s.istakCache.getPlayer(p.Size(), 1, true)

	pass, e := p.Move(tak.Move{Type: tak.Pass})
	pv, value, _ := player.Analyze(ctx, pass)

	var resp pb.IsPositionInTakResponse
	resp.InTak = value > ai.WinThreshold
	if resp.InTak {
		resp.TakMove = ptn.FormatMove(pv[0])
	}
	return &resp, nil

}

func (c *Command) Execute(ctx context.Context, flag *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	log.Printf("Listening on port %d", c.port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterTakticianServer(grpcServer, &server{})

	grpcServer.Serve(lis)
	return subcommands.ExitSuccess
}
