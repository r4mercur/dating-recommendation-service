package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

func FetchRecommendationsFromElastic(userId string) ([]string, error) {
	esClient := GetElasticClient()

	// More Like This query with exclusion of the same user
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"more_like_this": map[string]interface{}{
							"fields":          []string{"hobby", "interest"},
							"like":            []map[string]interface{}{{"_index": "users", "_id": userId}},
							"min_term_freq":   1,
							"max_query_terms": 12,
						},
					},
				},
				"must_not": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"id": userId,
						},
					},
				},
			},
		},
		"size": 10,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	log.Printf("Elasticsearch Query: %s", buf.String())

	res, err := esClient.Search(
		esClient.Search.WithContext(context.Background()),
		esClient.Search.WithIndex("users"),
		esClient.Search.WithBody(&buf),
		esClient.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %s", err)
		}
	}(res.Body)

	if res.IsError() {
		return nil, fmt.Errorf("error fetching recommendations: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	log.Printf("Elasticsearch Result: %v", result)

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	var recommendations []string
	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		recommendation := source["id"].(string)
		if recommendation != userId {
			recommendations = append(recommendations, recommendation)
		}
	}

	// Fallback strategy if no recommendations are found
	if len(recommendations) == 0 {
		fallbackQuery := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must_not": []map[string]interface{}{
						{
							"term": map[string]interface{}{
								"id": userId,
							},
						},
					},
				},
			},
			"size": 10,
		}

		var fallbackBuf bytes.Buffer
		if err := json.NewEncoder(&fallbackBuf).Encode(fallbackQuery); err != nil {
			return nil, err
		}

		log.Printf("Elasticsearch Fallback Query: %s", fallbackBuf.String())

		fallbackRes, err := esClient.Search(
			esClient.Search.WithContext(context.Background()),
			esClient.Search.WithIndex("users"),
			esClient.Search.WithBody(&fallbackBuf),
			esClient.Search.WithTrackTotalHits(true),
		)
		if err != nil {
			return nil, err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Printf("Error closing response body: %s", err)
			}
		}(fallbackRes.Body)

		if fallbackRes.IsError() {
			return nil, fmt.Errorf("error fetching fallback recommendations: %s", fallbackRes.String())
		}

		var fallbackResult map[string]interface{}
		if err := json.NewDecoder(fallbackRes.Body).Decode(&fallbackResult); err != nil {
			return nil, err
		}

		fallbackHits := fallbackResult["hits"].(map[string]interface{})["hits"].([]interface{})
		for _, hit := range fallbackHits {
			source := hit.(map[string]interface{})["_source"].(map[string]interface{})
			recommendation := source["id"].(string)
			if recommendation != userId {
				recommendations = append(recommendations, recommendation)
			}
		}
	}

	return recommendations, nil
}
