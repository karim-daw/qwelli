package main

import (
	"encoding/json"
	"fmt"

	"github.com/knights-analytics/hugot"
)

func check(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	// start a new session
	session, err := hugot.NewGoSession()
	// For XLA (requires go build tags "XLA" or "ALL"):
	// session, err := hugot.NewXLASession()
	// For ORT (requires go build tags "ORT" or "ALL"):
	// session, err := hugot.NewORTSession()
	// This looks for the onnxruntime.so library in its default path, e.g. /usr/lib/onnxruntime.so
	// If your onnxruntime.so is somewhere else, you can explicitly set it by using WithOnnxLibraryPath
	// session, err := hugot.NewORTSession(WithOnnxLibraryPath("/path/to/onnxruntime.so"))
	check(err)

	// A successfully created hugot session needs to be destroyed when you're done
	defer func(session *hugot.Session) {
		err := session.Destroy()
		check(err)
	}(session)

	// Download an embedding model (all-MiniLM-L6-v2 is a popular, lightweight embedding model)
	// For Google EmbeddingGemma, you would use: "KnightsAnalytics/embedding-gemma" (if available as ONNX)
	// note: if you compile your library with build flag NODOWNLOAD, this will exclude the downloader.
	modelPath, err := hugot.DownloadModel("KnightsAnalytics/all-MiniLM-L6-v2", "./models/", hugot.NewDownloadOptions())
	check(err)

	// Create the configuration for the feature extraction (embeddings) pipeline
	config := hugot.FeatureExtractionConfig{
		ModelPath:    modelPath,
		Name:         "embeddingPipeline",
		OnnxFilename: "model.onnx",
	}

	// Create the pipeline
	embeddingPipeline, err := hugot.NewPipeline(session, config)
	check(err)

	// Generate embeddings for a batch of strings
	texts := []string{"Hello world", "This is a test", "Embeddings are useful"}
	batchResult, err := embeddingPipeline.RunPipeline(texts)
	check(err)

	// Print the embeddings
	fmt.Printf("Generated embeddings for %d texts:\n", len(texts))
	for i, embedding := range batchResult.Embeddings {
		fmt.Printf("\nText %d: %q\n", i+1, texts[i])
		fmt.Printf("Embedding dimension: %d\n", len(embedding))
		previewLen := 5
		if len(embedding) < previewLen {
			previewLen = len(embedding)
		}
		fmt.Printf("First %d values: %v\n", previewLen, embedding[:previewLen])
	}

	// Optionally, print full result as JSON
	resultJSON, err := json.Marshal(batchResult)
	check(err)
	fmt.Printf("\n\nFull result (JSON):\n%s\n", string(resultJSON))
}

// to run: go run tests/test_embedding.go
