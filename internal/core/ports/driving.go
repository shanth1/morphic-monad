package ports

import "context"

type GatewayService interface {
	Ingest(ctx context.Context, tenantID string, data string) error
}

type RouterService interface {
	StartWorker(ctx context.Context) error
}
