package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/elastic/go-elasticsearch/v8"
	"recommandation.com/m/data"
	"recommandation.com/m/search"
)

const (
	UsersIndex = "users" // Index name constant
	BatchSize  = 100     // Batch size for bulk requests
	maxRetries = 3
	retryDelay = 2 * time.Second
)

func ImportUsersAndAddToElasticIndex() {
	if checkIfIndexHasData(UsersIndex) {
		log.Printf("Index '%s' already has data. Skipping import.", UsersIndex)
		return
	}

	log.Printf("Index '%s' is empty. Importing data...", UsersIndex)
	users := createFakeUsers(100_000)
	addUsersToElasticIndex(users)
	log.Println("Data import completed.")
}

func checkIfIndexHasData(indexName string) bool {
	esClient := search.GetElasticClient()

	res, err := esClient.Search(
		esClient.Search.WithIndex(indexName),
		esClient.Search.WithSize(0), // Only fetch metadata
	)
	if err != nil {
		log.Fatalf("Error checking if index '%s' has data: %s", indexName, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %s", err)
		}
	}(res.Body)

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Fatalf("Error parsing response body for index '%s': %s", indexName, err)
	}

	hits := result["hits"].(map[string]interface{})
	total := hits["total"].(map[string]interface{})["value"].(float64)

	return total > 0
}

func createFakeUsers(count int) []*data.User {
	gofakeit.Seed(0)

	var users []*data.User
	for i := 0; i < count; i++ {
		user := &data.User{
			ID:       gofakeit.UUID(),
			Name:     gofakeit.Name(),
			Email:    gofakeit.Email(),
			Interest: []string{gofakeit.RandomString([]string{"music", "sports", "movies", "books", "travel"}), gofakeit.RandomString([]string{"music", "sports", "movies", "books", "travel"})},
			Hobby:    []string{gofakeit.RandomString([]string{"music", "sports", "movies", "books", "travel"}), gofakeit.RandomString([]string{"music", "sports", "movies", "books", "travel"})},
			Age:      gofakeit.Number(18, 60),
			Address:  gofakeit.Address().Address,
		}
		users = append(users, user)
	}

	return users
}

func addUsersToElasticIndex(users []*data.User) {
	esClient := search.GetElasticClient()
	var wg sync.WaitGroup

	for i := 0; i < len(users); i += BatchSize {
		end := i + BatchSize
		if end > len(users) {
			end = len(users)
		}

		batch := users[i:end]
		wg.Add(1)
		go func(batch []*data.User) {
			defer wg.Done()
			if err := sendBulkRequest(esClient, batch); err != nil {
				log.Printf("Error sending bulk request: %s", err)
			}
			log.Printf("Indexed batch %d-%d of %d users", i+1, end, len(users))
		}(batch)
	}

	wg.Wait()
}

func sendBulkRequest(esClient *elasticsearch.Client, users []*data.User) error {
	var bulkRequest bytes.Buffer

	for _, user := range users {
		meta := fmt.Sprintf(`{"index":{"_index":"%s"}}`, UsersIndex)
		bulkRequest.WriteString(meta + "\n")

		userJSON, err := json.Marshal(user)
		if err != nil {
			return fmt.Errorf("error marshaling user to JSON: %w", err)
		}
		bulkRequest.Write(userJSON)
		bulkRequest.WriteString("\n")
	}

	for i := 0; i < maxRetries; i++ {
		res, err := esClient.Bulk(
			strings.NewReader(bulkRequest.String()),
			esClient.Bulk.WithIndex(UsersIndex),
		)
		if err != nil {
			return fmt.Errorf("error sending bulk request: %w", err)
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Printf("Error closing response body: %s", err)
			}
		}(res.Body)

		if res.IsError() {
			if res.StatusCode == 429 { // Too Many Requests
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("bulk request failed: %s", res.String())
		}

		return nil
	}

	return fmt.Errorf("max retries reached")
}
