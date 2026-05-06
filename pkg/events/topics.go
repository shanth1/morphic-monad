package events

// Topic strictly defines data bus channels
type Topic string

const (
	// IngressChannel - a single entry point for all new events into the platform
	TopicIngress Topic = "platform.events.ingress"

	// Worker Topics — channels for specific workers
	TopicTaskEmbed  Topic = "platform.tasks.embed"
	TopicTaskEngine Topic = "platform.tasks.engine"
	TopicTaskVision Topic = "platform.tasks.vision"
	TopicTaskChunk  Topic = "platform.tasks.chunk"

	// DLQ (Dead Letter Queue) - a channel for fatal errors
	TopicDLQ Topic = "platform.system.dlq"
)

// QueueGroup strictly defines consumer groups for load balancing
const (
	QueueGroupRouter   = "router_group"
	QueueGroupVision   = "vision_group"
	QueueGroupChunker  = "chunker_group"
	QueueGroupEmbedder = "embedder_group"
	QueueGroupEngine   = "engine_group"
)
