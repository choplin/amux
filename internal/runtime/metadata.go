package runtime

// Metadata represents runtime-specific metadata that can be serialized
type Metadata interface {
	// ToMap converts the metadata to a map for serialization
	ToMap() map[string]interface{}

	// RuntimeType returns the runtime type this metadata belongs to
	RuntimeType() string
}

// MetadataFromMap reconstructs metadata from a map based on runtime type
func MetadataFromMap(runtimeType string, data map[string]interface{}) Metadata {
	// This would be extended by each runtime implementation
	// For now, return a generic implementation
	return &genericMetadata{
		runtimeType: runtimeType,
		data:        data,
	}
}

// genericMetadata is a fallback implementation for unknown runtime types
type genericMetadata struct {
	runtimeType string
	data        map[string]interface{}
}

func (m *genericMetadata) ToMap() map[string]interface{} {
	return m.data
}

func (m *genericMetadata) RuntimeType() string {
	return m.runtimeType
}
