package events

type EventType string

const (
	// --- Ingestion & Claim Check ---
	EventIngestRequested EventType = "data.ingest.requested"

	// --- Search (Engine) ---
	EventSearchRequested EventType = "data.search.requested"
	EventSearchCompleted EventType = "data.search.completed"

	// --- Workers ---
	EventTaskVisionRequested EventType = "task.vision.requested"
	EventTaskVisionCompleted EventType = "task.vision.completed"
	EventTaskChunkRequested  EventType = "task.chunk.requested"
	EventTaskChunkCompleted  EventType = "task.chunk.completed"
	EventTaskEmbedRequested  EventType = "task.embed.requested"
	EventTaskEmbedCompleted  EventType = "task.embed.completed"
)
