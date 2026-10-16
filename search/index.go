package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// userIndexDefinition maps tags and gender as lowercase-normalized keywords so that
// "Music" and "music" match, and location as geo_point for distance filtering.
const userIndexDefinition = `{
  "settings": {
    "analysis": {
      "normalizer": {
        "lowercase": { "type": "custom", "filter": ["lowercase"] }
      }
    }
  },
  "mappings": {
    "properties": {
      "id":       { "type": "keyword" },
      "name":     { "type": "text" },
      "email":    { "type": "keyword" },
      "interest": { "type": "keyword", "normalizer": "lowercase" },
      "hobby":    { "type": "keyword", "normalizer": "lowercase" },
      "age":      { "type": "integer" },
      "gender":   { "type": "keyword", "normalizer": "lowercase" },
      "status":   { "type": "keyword", "normalizer": "lowercase" },
      "photo":    { "type": "keyword", "index": false },
      "location": { "type": "geo_point" },
      "address": {
        "properties": {
          "street":  { "type": "text" },
          "city":    { "type": "keyword" },
          "zip":     { "type": "keyword" },
          "country": { "type": "keyword" }
        }
      }
    }
  }
}`

// EnsureUserIndex creates the user index with its mapping, or verifies that an
// existing index can serve location-based queries.
func EnsureUserIndex(ctx context.Context, es *elasticsearch.Client, index string) error {
	res, err := es.Indices.Exists([]string{index}, es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("error checking index %q: %w", index, err)
	}
	closeBody(res)

	switch res.StatusCode {
	case http.StatusOK:
		log.Printf("The index %s already exists", index)
		return verifyLocationMapping(ctx, es, index)
	case http.StatusNotFound:
		return createUserIndex(ctx, es, index)
	default:
		return fmt.Errorf("error checking index %q: unexpected status %s", index, res.Status())
	}
}

func createUserIndex(ctx context.Context, es *elasticsearch.Client, index string) error {
	res, err := es.Indices.Create(
		index,
		es.Indices.Create.WithBody(strings.NewReader(userIndexDefinition)),
		es.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error creating index %q: %w", index, err)
	}
	defer closeBody(res)

	if res.IsError() {
		return responseError("create index "+index, res)
	}

	log.Printf("The index %s has been created", index)
	return nil
}

func verifyLocationMapping(ctx context.Context, es *elasticsearch.Client, index string) error {
	res, err := es.Indices.GetFieldMapping(
		[]string{"location"},
		es.Indices.GetFieldMapping.WithIndex(index),
		es.Indices.GetFieldMapping.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error reading mapping of index %q: %w", index, err)
	}
	defer closeBody(res)

	if res.IsError() {
		return responseError("read mapping of index "+index, res)
	}

	var mappings map[string]struct {
		Mappings map[string]struct {
			Mapping map[string]struct {
				Type string `json:"type"`
			} `json:"mapping"`
		} `json:"mappings"`
	}
	if err := json.NewDecoder(res.Body).Decode(&mappings); err != nil {
		return fmt.Errorf("error parsing mapping of index %q: %w", index, err)
	}

	for name, m := range mappings {
		if m.Mappings["location"].Mapping["location"].Type != "geo_point" {
			return fmt.Errorf(
				"index %q has no geo_point mapping for 'location' (it was probably created before location matching existed); "+
					"delete it with DELETE /%s and restart, or set ELASTIC_INDEX to a new name", name, name)
		}
	}
	return nil
}
