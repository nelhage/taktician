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
	return `serve`
}

func (c *Command) SetFlags(flags *flag.FlagSet) {
	flags.IntVar(&c.port, "port", 55430, "bind port")
}

type server struct {
	cache struct {
		sync.Mutex
		player  *ai.MinimaxAI
		cfg     ai.MinimaxConfig
		precise bool
	}
}

func (s *server) Analyze(ctx context.Context, req *pb.AnalyzeRequest) (*pb.AnalyzeResponse, error) {
	s.cache.Lock()
	defer s.cache.Unlock()

	p, e := ptn.ParseTPS(req.Position)
	if e != nil {
		return nil, e
	}

	if s.cache.cfg.Size != p.Size() || s.cache.cfg.Depth != int(req.Depth) || s.cache.precise != req.Precise {
		s.cache.cfg = ai.MinimaxConfig{
			Size:  p.Size(),
			Depth: int(req.Depth),
			Debug: 1,
		}
		if req.Precise {
			s.cache.cfg.MakePrecise()
		}
		s.cache.player = ai.NewMinimax(s.cache.cfg)
		s.cache.precise = req.Precise
	}

	var resp pb.AnalyzeResponse
	pv, value, _ := s.cache.player.Analyze(ctx, p)
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
