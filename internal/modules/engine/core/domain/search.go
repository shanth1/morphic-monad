package domain

// SearchQuery is a user request to the Engine
type SearchQuery struct {
	QueryText      string            `json:"query_text"`      // Request text
	TopK           int               `json:"top_k"`           // Number of results returned
	Filters        map[string]string `json:"filters"`         // Metadata filters
	UseHyDE        bool              `json:"use_hyde"`        // HyDE algorithm usage flag
	ConversationID string            `json:"conversation_id"` // Session ID (for agents with memory)
}

// Chunk represents a fragment of text for vectorization and searching.
type Chunk struct {
	ChunkID    string            `json:"chunk_id"`
	DocumentID string            `json:"document_id"`
	Text       string            `json:"text"`
	Tokens     int               `json:"tokens"`
	Metadata   map[string]string `json:"metadata"`
}
