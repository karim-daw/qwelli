package fileprocessor

// Registry manages file processors
type Registry struct {
	processors []FileProcessor
}

// NewRegistry creates a new file processor registry with default processors
func NewRegistry() *Registry {
	return &Registry{
		processors: []FileProcessor{
			NewPDFProcessor(),
			NewTextProcessor(),
		},
	}
}

// Register adds a new file processor to the registry
func (r *Registry) Register(processor FileProcessor) {
	r.processors = append(r.processors, processor)
}

// GetProcessor returns the first processor that can handle the given file type
func (r *Registry) GetProcessor(fileType string) FileProcessor {
	for _, processor := range r.processors {
		if processor.CanProcess(fileType) {
			return processor
		}
	}
	return nil
}

// DefaultRegistry is the default global registry
var DefaultRegistry = NewRegistry()

// GetProcessor returns a processor from the default registry
func GetProcessor(fileType string) FileProcessor {
	return DefaultRegistry.GetProcessor(fileType)
}
