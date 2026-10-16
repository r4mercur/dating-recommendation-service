package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"recommandation.com/m/data"
)

const (
	maxBulkRetries = 3
	bulkRetryDelay = 2 * time.Second
)

var ErrNotFound = errors.New("user not found")

type Hit struct {
	User  data.User
	Score float64
}

// UserStore reads and writes user documents. The document _id is always the user ID.
type UserStore struct {
	es    *elasticsearch.Client
	index string
}

func NewUserStore(es *elasticsearch.Client, index string) *UserStore {
	return &UserStore{es: es, index: index}
}

func (s *UserStore) Get(ctx context.Context, id string) (*data.User, error) {
	res, err := s.es.Get(s.index, id, s.es.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("error fetching user %s: %w", id, err)
	}
	defer closeBody(res)

	if res.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if res.IsError() {
		return nil, responseError("get user "+id, res)
	}

	var doc struct {
		Source data.User `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("error parsing user %s: %w", id, err)
	}
	return &doc.Source, nil
}

func (s *UserStore) Save(ctx context.Context, user *data.User) error {
	body, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("error marshaling user: %w", err)
	}

	res, err := s.es.Index(
		s.index,
		bytes.NewReader(body),
		s.es.Index.WithDocumentID(user.ID),
		s.es.Index.WithRefresh("wait_for"),
		s.es.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error adding user to index: %w", err)
	}
	defer closeBody(res)

	if res.IsError() {
		return responseError("index user "+user.ID, res)
	}
	return nil
}

// SaveAll indexes users with one bulk request and retries when Elasticsearch is overloaded.
func (s *UserStore) SaveAll(ctx context.Context, users []*data.User) error {
	var body bytes.Buffer
	for _, user := range users {
		meta, err := json.Marshal(map[string]any{"index": map[string]string{"_id": user.ID}})
		if err != nil {
			return fmt.Errorf("error marshaling bulk metadata: %w", err)
		}
		doc, err := json.Marshal(user)
		if err != nil {
			return fmt.Errorf("error marshaling user %s: %w", user.ID, err)
		}
		body.Write(meta)
		body.WriteByte('\n')
		body.Write(doc)
		body.WriteByte('\n')
	}

	for attempt := 1; attempt <= maxBulkRetries; attempt++ {
		retry, err := s.bulk(ctx, body.Bytes())
		if !retry {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(bulkRetryDelay):
		}
	}
	return fmt.Errorf("bulk request failed after %d retries", maxBulkRetries)
}

func (s *UserStore) bulk(ctx context.Context, body []byte) (retry bool, err error) {
	res, err := s.es.Bulk(
		bytes.NewReader(body),
		s.es.Bulk.WithIndex(s.index),
		s.es.Bulk.WithContext(ctx),
	)
	if err != nil {
		return false, fmt.Errorf("error sending bulk request: %w", err)
	}
	defer closeBody(res)

	if res.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}
	if res.IsError() {
		return false, responseError("bulk request", res)
	}

	var result struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			ID    string `json:"_id"`
			Error *struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("error parsing bulk response: %w", err)
	}
	if !result.Errors {
		return false, nil
	}

	var errs []error
	for _, item := range result.Items {
		for _, op := range item {
			if op.Error != nil {
				errs = append(errs, fmt.Errorf("user %s: %s: %s", op.ID, op.Error.Type, op.Error.Reason))
			}
		}
	}
	return false, fmt.Errorf("bulk request had %d failed items: %w", len(errs), errors.Join(errs...))
}

func (s *UserStore) Count(ctx context.Context) (int, error) {
	res, err := s.es.Count(s.es.Count.WithIndex(s.index), s.es.Count.WithContext(ctx))
	if err != nil {
		return 0, fmt.Errorf("error counting users: %w", err)
	}
	defer closeBody(res)

	if res.IsError() {
		return 0, responseError("count users", res)
	}

	var result struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("error parsing count response: %w", err)
	}
	return result.Count, nil
}

// Search runs a raw search request body and returns the matching users ordered by score.
func (s *UserStore) Search(ctx context.Context, body map[string]any) ([]Hit, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling search request: %w", err)
	}

	res, err := s.es.Search(
		s.es.Search.WithIndex(s.index),
		s.es.Search.WithBody(bytes.NewReader(payload)),
		s.es.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("error searching users: %w", err)
	}
	defer closeBody(res)

	if res.IsError() {
		return nil, responseError("search users", res)
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Score  float64   `json:"_score"`
				Source data.User `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing search response: %w", err)
	}

	hits := make([]Hit, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		hits = append(hits, Hit{User: h.Source, Score: h.Score})
	}
	return hits, nil
}
