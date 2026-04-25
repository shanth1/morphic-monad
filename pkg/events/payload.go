package events

// Structures that are passed inside the Data field of Envelope

// ClaimCheckPayload is a "receipt" (Claim Check) for a heavy file
// This structure is serialized into the Data field of Envelope struct on EventDocumentUploaded type
type ClaimCheckPayload struct {
	DocumentID string            `json:"document_id"` // Unique ID of the document in the database
	BlobURI    string            `json:"blob_uri"`    // Link to S3/MinIO
	MimeType   string            `json:"mime_type"`   // File type (application/pdf, video/mp4)
	SizeBytes  int64             `json:"size_bytes"`
	Metadata   map[string]string `json:"metadata"`
}

// TextPayload is passed to handle raw text
type TextPayload struct {
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata"`
}
