package search

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"recommandation.com/m/config"
)

func NewClient(cfg config.Config) (*elasticsearch.Client, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{cfg.ElasticURL},
		Username:  cfg.ElasticUser,
		Password:  cfg.ElasticPassword,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.ElasticInsecureSkipVerify},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating elasticsearch client: %w", err)
	}
	return client, nil
}

func closeBody(res *esapi.Response) {
	if err := res.Body.Close(); err != nil {
		log.Printf("error closing response body: %s", err)
	}
}

func responseError(op string, res *esapi.Response) error {
	body, _ := io.ReadAll(res.Body)
	return fmt.Errorf("%s: elasticsearch returned %s: %s", op, res.Status(), body)
}
