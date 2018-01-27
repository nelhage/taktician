package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"

	"github.com/nelhage/taktician/ai"
	"github.com/nelhage/taktician/pb/tak/proto"
	"github.com/nelhage/taktician/ptn"
)

type server struct {
	sync.Mutex

	cache       *ai.MinimaxAI
	cacheConfig ai.MinimaxConfig
}

func (s *server) Analyze(ctx context.Context, req *pb.AnalyzeRequest) (*pb.AnalyzeResponse, error) {
	s.Lock()
	defer s.Unlock()

	p, e := ptn.ParseTPS(req.Position)
	if e != nil {
		return nil, e
	}

	if s.cacheConfig.Size != p.Size() || s.cacheConfig.Depth != int(req.Depth) {
		s.cacheConfig = ai.MinimaxConfig{
			Size:  p.Size(),
			Depth: int(req.Depth),
			Debug: 1,
		}
		s.cache = ai.NewMinimax(s.cacheConfig)
	}

	var resp pb.AnalyzeResponse
	pv, value, _ := s.cache.Analyze(ctx, p)
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
