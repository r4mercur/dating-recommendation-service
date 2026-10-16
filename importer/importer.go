package importer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"recommandation.com/m/data"
	"recommandation.com/m/search"
)

const (
	BatchSize          = 100 // Batch size for bulk requests
	maxParallelBatches = 4
)

var ErrIndexNotEmpty = errors.New("index already contains users")

type Importer struct {
	store       *search.UserStore
	userService *UserServiceClient
}

// New creates an importer. userService may be nil to skip mirroring fake users.
func New(store *search.UserStore, userService *UserServiceClient) *Importer {
	return &Importer{store: store, userService: userService}
}

func (i *Importer) ImportUser(ctx context.Context, user *data.User) error {
	normalize(user)
	return i.store.Save(ctx, user)
}

// ImportFakeUsers fills an empty index with count generated users and returns
// ErrIndexNotEmpty if the index already has data.
func (i *Importer) ImportFakeUsers(ctx context.Context, count int) error {
	existing, err := i.store.Count(ctx)
	if err != nil {
		return err
	}
	if existing > 0 {
		return fmt.Errorf("%w (%d users)", ErrIndexNotEmpty, existing)
	}

	log.Printf("Index is empty. Importing %d fake users...", count)
	users := createFakeUsers(count)

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
		sem  = make(chan struct{}, maxParallelBatches)
	)
	for start := 0; start < len(users); start += BatchSize {
		end := min(start+BatchSize, len(users))
		batch := users[start:end]

		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if err := i.importBatch(ctx, batch); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("batch %d-%d: %w", start+1, end, err))
				mu.Unlock()
				return
			}
			log.Printf("Indexed batch %d-%d of %d users", start+1, end, len(users))
		}()
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return err
	}
	log.Println("Data import completed.")
	return nil
}

func (i *Importer) importBatch(ctx context.Context, users []*data.User) error {
	if i.userService != nil {
		for _, user := range users {
			if err := i.userService.CreateUser(ctx, user); err != nil {
				log.Printf("Error posting user %s to user service: %s", user.ID, err)
			}
		}
	}
	return i.store.SaveAll(ctx, users)
}

func normalize(user *data.User) {
	user.Gender = strings.ToUpper(strings.TrimSpace(user.Gender))
	user.Status = strings.ToUpper(strings.TrimSpace(user.Status))
}
