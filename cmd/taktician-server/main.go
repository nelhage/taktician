package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/pb/tak/proto"
	"github.com/nelhage/taktician/ptn"
)

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

func main() {
	var (
		port = flag.Int("port", 55430, "bind port")
	)

	flag.Parse()
	log.Printf("Listening on port %d", *port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterTakticianServer(grpcServer, &server{})

	grpcServer.Serve(lis)
}
