package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"recommandation.com/m/data"
)

// UserServiceClient mirrors generated users to the user service so both services share the same IDs.
type UserServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewUserServiceClient(baseURL string) *UserServiceClient {
	return &UserServiceClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *UserServiceClient) CreateUser(ctx context.Context, user *data.User) error {
	userServiceReq := data.UserServiceRequest{
		ReferenceID: user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Password:    "",
		Interests:   user.Interests,
		Hobbies:     user.Hobbies,
		Age:         user.Age,
		Address:     user.Address,
		Location:    user.Location,
		Gender:      user.Gender,
		Status:      user.Status,
		Photo:       user.Photo,
	}

	userJSON, err := json.Marshal(userServiceReq)
	if err != nil {
		return fmt.Errorf("error marshaling user to JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/user?useDefaultPW=true&addNotToSearchIndex=true", bytes.NewReader(userJSON))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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
