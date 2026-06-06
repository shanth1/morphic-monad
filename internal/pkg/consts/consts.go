package consts

const (
	AppName string = "morphic-monad"

	ServiceMonolith string = "monolith"

	// Main Modules
	ServiceGateway string = "gateway"
	ServiceRouter  string = "router"
	ServiceEngine  string = "engine"

	// Workers
	ServiceEmbedder string = "embedder"

	// Components
	ComponentNATSClient = "nats_client"
	ComponentHTTPServer = "http_server"
)
