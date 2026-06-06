package events

// Topic strictly defines data bus channels
type Topic string

const (
	// IngressChannel - a single entry point for all new events into the platform
	TopicIngress Topic = "platform.events.ingress"

	// Worker Topics — channels for specific workers
	TopicTaskEmbed  Topic = "platform.tasks.embed"
	TopicTaskEngine Topic = "platform.tasks.engine"

	// DLQ (Dead Letter Queue) - a channel for fatal errors
	TopicDLQ Topic = "platform.system.dlq"
)

// QueueGroup strictly defines consumer groups for load balancing
const (
	QueueGroupRouter   string = "router_cluster"
	QueueGroupEmbedder string = "worker_embedder_cluster"
	QueueGroupEngine   string = "engine_cluster"
)
