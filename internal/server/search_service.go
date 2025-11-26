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
	folderPath := req.GetFolderPath()

	// search for matches of files given a folder path
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read folder: %v", err)
	}
	// convert query to lower case for case insensitive search
	searchQueryLower := strings.ToLower(searchQuery)

	results := []string{}
	for _, file := range files {
		// check if string in query is in file name
		fileLower := strings.ToLower(file.Name())
		if strings.Contains(fileLower, searchQueryLower) {
			results = append(results, filepath.Join(folderPath, file.Name()))
		}
	}

	// if no matches were found in the file title, we parse the content of the file to find matches
	if len(results) == 0 {

		for _, file := range files {
			content, err := os.ReadFile(filepath.Join(folderPath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read file: %v", err)
			}
			if strings.Contains(strings.ToLower(string(content)), searchQueryLower) {
				results = append(results, filepath.Join(folderPath, file.Name()))
			}
		}
	}
	return &pb.SearchResponse{Results: results}, nil
}
