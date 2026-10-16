package recommendation

import (
	"context"
	"math"
	"strings"

	"recommandation.com/m/data"
	"recommandation.com/m/search"
)

const (
	DefaultRadiusKm = 50.0
	DefaultLimit    = 10
)

type Options struct {
	RadiusKm float64
	Limit    int
	MinAge   int
	MaxAge   int
	Gender   string
}

func (o Options) withDefaults() Options {
	if o.RadiusKm <= 0 {
		o.RadiusKm = DefaultRadiusKm
	}
	if o.Limit <= 0 {
		o.Limit = DefaultLimit
	}
	return o
}

// Recommendation is the public view of a suggested partner. It deliberately leaves out
// contact data like email and the exact address.
type Recommendation struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Age             int      `json:"age"`
	Gender          string   `json:"gender,omitempty"`
	City            string   `json:"city,omitempty"`
	Photo           string   `json:"photo,omitempty"`
	DistanceKm      *float64 `json:"distanceKm,omitempty"`
	SharedInterests []string `json:"sharedInterests"`
	SharedHobbies   []string `json:"sharedHobbies"`
	Score           float64  `json:"score"`
}

type Store interface {
	Get(ctx context.Context, id string) (*data.User, error)
	Search(ctx context.Context, body map[string]any) ([]search.Hit, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Recommend returns partner suggestions for the given user. It returns search.ErrNotFound
// if the user is not in the index.
func (s *Service) Recommend(ctx context.Context, userID string, opts Options) ([]Recommendation, error) {
	user, err := s.store.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	hits, err := s.store.Search(ctx, buildQuery(user, opts.withDefaults()))
	if err != nil {
		return nil, err
	}

	recommendations := make([]Recommendation, 0, len(hits))
	for _, hit := range hits {
		candidate := hit.User
		rec := Recommendation{
			ID:              candidate.ID,
			Name:            candidate.Name,
			Age:             candidate.Age,
			Gender:          candidate.Gender,
			City:            candidate.Address.City,
			Photo:           candidate.Photo,
			SharedInterests: intersect(user.Interests, candidate.Interests),
			SharedHobbies:   intersect(user.Hobbies, candidate.Hobbies),
			Score:           hit.Score,
		}
		if user.Location != nil && candidate.Location != nil {
			distance := math.Round(distanceKm(*user.Location, *candidate.Location)*10) / 10
			rec.DistanceKm = &distance
		}
		recommendations = append(recommendations, rec)
	}
	return recommendations, nil
}

// intersect returns the tags of b that also occur in a, compared case-insensitively.
func intersect(a, b []string) []string {
	wanted := make(map[string]bool, len(a))
	for _, tag := range a {
		wanted[strings.ToLower(strings.TrimSpace(tag))] = true
	}

	shared := []string{}
	seen := map[string]bool{}
	for _, tag := range b {
		key := strings.ToLower(strings.TrimSpace(tag))
		if wanted[key] && !seen[key] {
			seen[key] = true
			shared = append(shared, tag)
		}
	}
	return shared
}

const earthRadiusKm = 6371.0

// distanceKm returns the great-circle distance between two points (haversine formula).
func distanceKm(a, b data.GeoPoint) float64 {
	lat1, lat2 := a.Lat*math.Pi/180, b.Lat*math.Pi/180
	dLat := lat2 - lat1
	dLon := (b.Lon - a.Lon) * math.Pi / 180

	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(h))
}
