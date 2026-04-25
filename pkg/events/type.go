package events

type EventType string

const (
	// --- Ingestion & Claim Check ---

	// EventDocumentUploaded is triggered by Gateway after saving a heavy file to BlobStore (Claim Check)
	EventDocumentUploaded EventType = "document.uploaded"

	// --- Pipelines ---

	EventPipelineStarted   EventType = "pipeline.started"
	EventPipelineCompleted EventType = "pipeline.completed"
	EventPipelineFailed    EventType = "pipeline.failed"

	// --- Workers: OCR ---

	EventTaskOCRPending   EventType = "task.ocr.pending"
	EventTaskOCRCompleted EventType = "task.ocr.completed"
	EventTaskOCRFailed    EventType = "task.ocr.failed"

	// --- Workers: Embeddings & VectorDB ---

	EventTaskEmbedPending   EventType = "task.embed.pending"
	EventTaskEmbedCompleted EventType = "task.embed.completed"
	EventTaskVectorUpsert   EventType = "task.vector.upsert" // Команда на сохранение в Qdrant
)
