package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func ImportSingleUserAndAddToElasticIndex(user *data.User) error {
	esClient := search.GetElasticClient()

	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("error when marshaling user: %w", err)
	}

	res, err := esClient.Index(
		UsersIndex,
		strings.NewReader(string(userJSON)),
		esClient.Index.WithDocumentID(user.ID),
		esClient.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("error when trying to add user to index: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("error when closing request body: %s", err)
		}
	}(res.Body)

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch-error: %s", string(body))
	}

	return nil
}

func ImportUsersAndAddToElasticIndex() {
	if checkIfIndexHasData(UsersIndex) {
		log.Printf("Index '%s' already has data. Skipping import.", UsersIndex)
		return
	}

	log.Printf("Index '%s' is empty. Importing data...", UsersIndex)
	users := createFakeUsers(1000)
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
		fakeAddr := gofakeit.Address()
		user := &data.User{
			ID:       gofakeit.UUID(),
			Name:     gofakeit.Name(),
			Email:    gofakeit.Email(),
			Interest: []string{gofakeit.RandomString([]string{"music", "sports", "movies", "books", "travel"}), gofakeit.RandomString([]string{"music", "sports", "movies", "books", "travel"})},
			Hobby:    []string{gofakeit.RandomString([]string{"music", "sports", "movies", "books", "travel"}), gofakeit.RandomString([]string{"music", "sports", "movies", "books", "travel"})},
			Age:      gofakeit.Number(18, 60),
			Address: data.Address{
				Street:  fakeAddr.Street,
				City:    fakeAddr.City,
				Zip:     fakeAddr.Zip,
				Country: fakeAddr.Country,
			},
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

func postUserToUserService(user *data.User) error {
	userServiceReq := data.UserServiceRequest{
		ReferenceID: user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Password:    "",
		Interests:   user.Interest,
		Hobbies:     user.Hobby,
		Age:         user.Age,
		Address: data.Address{
			Street:  user.Address.Street,
			City:    user.Address.City,
			Zip:     user.Address.Zip,
			Country: user.Address.Country,
		},
		Gender: strings.ToUpper(gofakeit.Gender()),
		Status: "ACTIVE",
		Photo:  "https://example.com/photo.jpg",
	}

	userJSON, err := json.Marshal(userServiceReq)
	if err != nil {
		return fmt.Errorf("error marshaling user to JSON: %w", err)
	}

	resp, err := http.Post("http://localhost:8080/api/user?useDefaultPW=true", "application/json", bytes.NewBuffer(userJSON))
	if err != nil {
		return fmt.Errorf("error sending POST request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %s", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("user service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func sendBulkRequest(esClient *elasticsearch.Client, users []*data.User) error {
	var bulkRequest bytes.Buffer

	// send all to user service
	for _, user := range users {
		if err := postUserToUserService(user); err != nil {
			log.Printf("Error posting user %s to user service: %s", user.ID, err)
		}
	}

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
