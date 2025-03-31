package search

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
)

var esClient *elasticsearch.Client

func InitElasticClient(user string, password string) {
	config := elasticsearch.Config{
		Addresses: []string{
			"https://localhost:9200",
		},
		Username: user,
		Password: password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	client, err := elasticsearch.NewClient(config)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	esClient = client
	createIndexIfNotExists(esClient, "users")
}

func GetElasticClient() *elasticsearch.Client {
	return esClient
}

func createIndexIfNotExists(es *elasticsearch.Client, indexName string) {
	res, err := es.Indices.Exists([]string{indexName})
	if err != nil {
		log.Fatalf("The index %s could not be checked for existence: %s", indexName, err)
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		log.Printf("The index %s already exists", indexName)
		return
	}

	if res.StatusCode == http.StatusNotFound {
		createIndex(es, indexName)
		return
	}
}

func createIndex(es *elasticsearch.Client, indexName string) {
	res, err := es.Indices.Create(indexName)
	if err != nil {
		log.Fatalf("The index %s could not be created: %s", indexName, err)
	}

	defer res.Body.Close()

	log.Printf("The index %s has been created", indexName)
}
