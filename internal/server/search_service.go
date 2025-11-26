package server

import (
	"context"

	pb "qwelli/api/gen/go/qwelli/v1"
)

type searchService struct {
	pb.UnimplementedSearchServiceServer
}

func NewSearchService() pb.SearchServiceServer {
	return &searchService{}
}

func (s *searchService) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	// simple dummy response for now
	results := []string{
		"Result 1 for: " + req.GetQuery(),
		"Result 2 for: " + req.GetQuery(),
	}
	return &pb.SearchResponse{
		Results: results,
	}, nil
}
