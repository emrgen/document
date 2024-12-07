package queue

import (
	"context"

	"github.com/emrgen/document/internal/model"
)

var DocumentUpdateCacheQueue = "document:update:queue"
var DocumentUpdateDatabaseQueue = "document:sync:queue"

type DocumentQueue interface {
	// PublishChange appends a document change to the queue.
	PublishChange(ctx context.Context, change *model.Document) error
	SubscribeUpdateCacheQueue(ctx context.Context) (<-chan *model.Document, error)
	SubscribeUpdateDatabaseQueue(ctx context.Context) (<-chan *model.Document, error)
}
