package events

type EventType string

const (
	// --- Ingestion & Claim Check ---
	EventDocumentUploaded EventType = "document.uploaded"

	// --- Search ---
	EventSearchRequested EventType = "document.search.requested"

	// --- Workers: Embeddings & VectorDB ---
	EventTaskEmbedCompleted EventType = "task.embed.completed"

	// --- Engine ---
	EventSearchCompleted EventType = "engine.search.completed"
)
