package events

type EventType string

const (
	// --- Ingestion & Claim Check ---
	EventIngestRequested EventType = "data.ingest.requested"

	// --- Search ---
	EventSearchRequested EventType = "data.search.requested"

	// --- Workers: Embeddings & VectorDB ---
	EventTaskEmbedCompleted EventType = "task.embed.completed"

	// --- Engine ---
	EventSearchCompleted EventType = "engine.search.completed"
)
