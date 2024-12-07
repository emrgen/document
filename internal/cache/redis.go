package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/emrgen/document/internal/compress"
	"github.com/emrgen/document/internal/model"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const (
	documentUpdateQueue         = "document:update:queue"
	documentSyncQueue           = "document:sync:queue"
	documentUpdatedVersionHash  = "document:updated:version"
	documentReadOnlyVersionHash = "document:read:only:version"
	documentVersionHash         = "document:version"
)

func documentUpdateKey(id string) string {
	return "document:update:" + id
}

func documentKey(id string) string {
	return "document:" + id
}

func documentSyncKey(id string) string {
	return "document:sync:" + id
}

var _ DocumentCache = (*RedisDocumentCache)(nil)

type RedisDocumentCache struct {
	client  *redis.Client
	encoder compress.Compress
}

func NewRedisDocumentCache() *RedisDocumentCache {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set
		DB:       0,  // Use default DB
		Protocol: 2,  // Connection protocol
	})

	return &RedisDocumentCache{client: client, encoder: compress.NewGZip()}
}

// func (r *RedisDocumentCache) Sync(ctx context.Context, count int64, db *gorm.DB) error {
// 	// get count number of updates from the queue
// 	updates := r.client.ZRange(ctx, documentUpdateQueue, 0, count)
// 	if updates.Err() != nil {
// 		return updates.Err()
// 	}

// 	for _, update := range updates.Val() {
// 		document := &model.Document{}

// 		err := json.Unmarshal([]byte(update), document)
// 		if err != nil {
// 			return err
// 		}

// 		err = document.UpdateChanges(db)
// 		if err != nil {
// 			return err
// 		}

// 		// remove the update from the queue after it is processed
// 		pop := r.client.LPop(ctx, documentUpdateQueue)
// 		if pop.Err() != nil {
// 			return pop.Err()
// 		}

// 		// sleep for a while to avoid overloading the database
// 		time.Sleep(time.Microsecond * 100)
// 	}

// 	return nil
// }

func (r *RedisDocumentCache) GetDocumentVersion(ctx context.Context, id uuid.UUID, mode GetDocumentMode) (int64, error) {
	if mode == GetDocumentModeView {
		updated := r.client.HGet(ctx, documentReadOnlyVersionHash, id.String())
		if updated.Err() != nil {
			return 0, updated.Err()
		}

		version, err := strconv.ParseInt(updated.Val(), 10, 64)
		if err != nil {
			return 0, err
		}

		return version, nil
	}

	updated := r.client.HGet(ctx, documentUpdatedVersionHash, id.String())
	if updated.Err() != nil {
		return 0, updated.Err()
	}

	version, err := strconv.ParseInt(updated.Val(), 10, 64)
	if err != nil {
		return 0, err
	}

	return version, nil
}

func (r *RedisDocumentCache) GetDocument(ctx context.Context, id uuid.UUID, mode GetDocumentMode) (*model.Document, error) {
	if mode == GetDocumentModeView {
		return r.getReadOnlyDocument(ctx, id)
	}

	return r.getUpdatedDocument(ctx, id)
}

func (r *RedisDocumentCache) getReadOnlyDocument(ctx context.Context, id uuid.UUID) (*model.Document, error) {
	// get the document from the cache
	res := r.client.Get(ctx, documentKey(id.String()))
	if res.Err() != nil {
		if errors.Is(res.Err(), redis.Nil) {
			return nil, nil
		} else {
			return nil, res.Err()
		}
	}

	doc := &model.Document{}

	buf, err := res.Bytes()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(buf, doc)

	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (r *RedisDocumentCache) getUpdatedDocument(ctx context.Context, id uuid.UUID) (*model.Document, error) {
	updates := r.client.LRange(ctx, documentKey(id.String()), 0, -1)
	if updates.Err() != nil {
		return nil, updates.Err()
	}

	document := &model.Document{
		ID:      id.String(),
		Content: "",
		Parts:   make([]string, 0),
	}

	for _, doc := range updates.Val() {
		var update model.Document
		err := json.Unmarshal([]byte(doc), &update)
		if err != nil {
			return nil, err
		}

		if update.Content != "" {
			document.Content = update.Content
			document.Parts = make([]string, 0)
		} else {
			document.Parts = append(document.Parts, update.Parts...)
		}
	}

	// merge the parts into the document content
	// document.Content = document.MergeParts()

	return document, nil
}

func (r *RedisDocumentCache) SetDocument(ctx context.Context, id uuid.UUID, doc *model.Document) error {
	// encode document to json
	marshal, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	_, err = r.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		if err := p.Set(ctx, documentKey(id.String()), marshal, time.Hour).Err(); err != nil {
			return err
		}

		if err := p.HSet(ctx, documentReadOnlyVersionHash, id.String(), doc.Version).Err(); err != nil {
			return err
		}

		return nil
	})

	return err
}

// used by job to sync the document updates to the cache
func (r *RedisDocumentCache) UpdateDocument(ctx context.Context, id uuid.UUID, doc *model.Document) error {
	// encode document to json
	marshal, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	// TODO: use real queue like kafka or rabbitmq to handle this
	// push the document to the update queue
	pushed := r.client.RPush(ctx, documentUpdateQueue, marshal)
	if pushed.Err() != nil {
		return pushed.Err()
	}

	documents := []string{string(marshal)}

	// append to the document update queue, a sync will be triggered later
	updated := r.client.HGet(ctx, documentVersionHash, id.String())
	if updated.Err() != nil {
		return updated.Err()
	}

	version, err := strconv.ParseFloat(updated.Val(), 64)
	if err != nil {
		return err
	}

	if version >= float64(doc.Version) {
		logrus.Warn("document update failed because the document is updated by another user")
		return errors.New("document is updated by another user, please refresh")
	}

	updateLen := r.client.LLen(ctx, documentUpdateKey(id.String()))
	if updateLen.Err() != nil {
		return updateLen.Err()
	}

	// update the document in the cache
	_, err = r.client.TxPipelined(ctx, func(tx redis.Pipeliner) error {
		if updateLen.Val() > 5 {
			// remove the oldest update
			updates := tx.LRange(ctx, documentUpdateKey(id.String()), 0, -1)
			if updates.Err() != nil {
				return updates.Err()
			}

			document := &model.Document{
				ID:      id.String(),
				Content: "",
				Parts:   make([]string, 0),
			}

			for _, doc := range updates.Val() {
				var update model.Document
				err := json.Unmarshal([]byte(doc), &update)
				if err != nil {
					return err
				}

				if update.Content != "" {
					document.Content = update.Content
					document.Parts = make([]string, 0)
				} else {
					document.Parts = append(document.Parts, update.Parts...)
				}
			}

			base, err := json.Marshal(document)
			if err != nil {
				return err
			}

			documents = []string{string(base), string(marshal)}
		}

		// convert documents to []interface{}
		docs := make([]interface{}, len(documents))
		for i, doc := range documents {
			docs[i] = doc
		}
		push := tx.RPush(ctx, documentUpdateKey(id.String()), docs...)
		if push.Err() != nil {
			return push.Err()
		}

		set := tx.HSet(ctx, documentUpdatedVersionHash, id.String(), doc.Version)
		if set.Err() != nil {
			return set.Err()
		}

		// keep the document in the cache for a while
		exp := tx.Expire(ctx, documentKey(id.String()), time.Hour)
		if exp.Err() != nil {
			return exp.Err()
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *RedisDocumentCache) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	return r.client.Del(ctx, documentKey(id.String())).Err()
}
