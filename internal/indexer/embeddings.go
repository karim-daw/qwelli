package indexer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
)

type Embedder struct {
	session  *hugot.Session
	pipeline *pipelines.FeatureExtractionPipeline
	dim      int
}

func NewEmbedder(modelName string, modelDir string, expectedDim int) (*Embedder, error) {
	session, err := hugot.NewGoSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Ensure model directory exists
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		session.Destroy()
		return nil, fmt.Errorf("failed to create model directory: %w", err)
	}

	expectedModelPath := filepath.Join(modelDir, modelName)
	expectedOnnxFile := filepath.Join(expectedModelPath, "model.onnx")

	var modelPath string
	if _, err := os.Stat(expectedOnnxFile); os.IsNotExist(err) {
		// Model doesn't exist, download it
		modelPath, err = hugot.DownloadModel(modelName, modelDir, hugot.NewDownloadOptions())
		if err != nil {
			session.Destroy()
			return nil, fmt.Errorf("failed to download model: %w", err)
		}
	} else {
		// Model already exists, use the expected path
		modelPath = expectedModelPath
	}

	config := hugot.FeatureExtractionConfig{
		ModelPath:    modelPath,
		Name:         "embeddingPipeline",
		OnnxFilename: "model.onnx",
	}

	pipeline, err := hugot.NewPipeline(session, config)
	if err != nil {
		session.Destroy()
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	return &Embedder{
		session:  session,
		pipeline: pipeline,
		dim:      expectedDim,
	}, nil
}

func (e *Embedder) Embed(text string) ([]float32, error) {
	result, err := e.pipeline.RunPipeline([]string{text})
	if err != nil {
		return nil, err
	}
	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return result.Embeddings[0], nil
}

func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	result, err := e.pipeline.RunPipeline(texts)
	if err != nil {
		return nil, err
	}
	return result.Embeddings, nil
}

func (e *Embedder) Close() error {
	if e.session != nil {
		return e.session.Destroy()
	}
	return nil
}

func GetModelPath(modelName string) string {
	homeDir, _ := os.UserHomeDir()
	if modelName == "" {
		return filepath.Join(homeDir, ".qwelli", "models")
	}
	return filepath.Join(homeDir, ".qwelli", "models", modelName)
}
