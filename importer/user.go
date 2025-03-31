package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/elastic/go-elasticsearch/v8"
	"recommandation.com/m/data"
	"recommandation.com/m/search"
)

const (
	UsersIndex = "users" // Index name constant
	BatchSize  = 100     // Batch size for bulk requests
)

func ImportUsersAndAddToElasticIndex() {
	if checkIfIndexHasData(UsersIndex) {
		log.Printf("Index '%s' already has data. Skipping import.", UsersIndex)
		return
	}

	log.Printf("Index '%s' is empty. Importing data...", UsersIndex)
	users := createFakeUsers(1000) // Generate 1000 fake users
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
	defer res.Body.Close()

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

	for i := 0; i < len(users); i += BatchSize {
		end := i + BatchSize
		if end > len(users) {
			end = len(users)
		}

		batch := users[i:end]
		if err := sendBulkRequest(esClient, batch); err != nil {
			log.Fatalf("Error sending bulk request: %s", err)
		}

		log.Printf("Indexed batch %d-%d of %d users", i+1, end, len(users))
	}
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

	res, err := esClient.Bulk(
		strings.NewReader(bulkRequest.String()),
		esClient.Bulk.WithIndex(UsersIndex),
	)
	if err != nil {
		return fmt.Errorf("error sending bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk request failed: %s", res.String())
	}

	return nil
}
