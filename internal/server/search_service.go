package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pb "qwelli/api/gen/go/qwelli/v1"
)

type searchService struct {
	pb.UnimplementedSearchServiceServer
}

func NewSearchService() pb.SearchServiceServer {
	return &searchService{}
}

func (s *searchService) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {

	// user query to search for in files
	searchQuery := req.GetQuery()

	// search for matches of files given a folder path
	files, err := os.ReadDir("tests/test_folder")
	if err != nil {
		return nil, fmt.Errorf("failed to read folder: %v", err)
	}

	// just return the files for now
	results := []string{}
	for _, file := range files {
		// check if string in query is in file name
		// first conver to lower
		searchQueryLower := strings.ToLower(searchQuery)
		fileLower := strings.ToLower(file.Name())
		if strings.Contains(fileLower, searchQueryLower) {
			results = append(results, filepath.Join("tests/test_folder", file.Name()))
		}
	}

	return &pb.SearchResponse{
		Results: results,
	}, nil
}
