package events

// Topic strictly defines data bus channels
type Topic string

const (
	// IngressChannel - a single entry point for all new events into the platform (listens to Router)
	TopicIngress Topic = "platform.events.ingress"

	// Worker Topics — channels for specific workers (listened to by Workers)
	TopicTaskOCR   Topic = "platform.tasks.ocr"
	TopicTaskEmbed Topic = "platform.tasks.embed"

	// DLQ (Dead Letter Queue) - a channel for fatal errors
	TopicDLQ Topic = "platform.system.dlq"
)

// QueueGroup strictly defines consumer groups for load balancing
// If two router instances share the same QueueGroup, NATS will only deliver the message to one of them
const (
	QueueGroupRouter string = "router_cluster"
	QueueGroupOCR    string = "worker_ocr_cluster"
)
