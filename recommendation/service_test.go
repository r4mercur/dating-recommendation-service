package recommendation

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"recommandation.com/m/data"
	"recommandation.com/m/search"
)

var (
	berlin  = data.GeoPoint{Lat: 52.52, Lon: 13.405}
	potsdam = data.GeoPoint{Lat: 52.3906, Lon: 13.0645}
	hamburg = data.GeoPoint{Lat: 53.5511, Lon: 9.9937}
)

type fakeStore struct {
	users       map[string]*data.User
	hits        []search.Hit
	lastRequest map[string]any
}

func (f *fakeStore) Get(_ context.Context, id string) (*data.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, search.ErrNotFound
}

func (f *fakeStore) Search(_ context.Context, body map[string]any) ([]search.Hit, error) {
	f.lastRequest = body
	return f.hits, nil
}

func TestBuildQueryWithLocation(t *testing.T) {
	user := &data.User{
		ID:        "u1",
		Interests: []string{"Music", "music", "travel"},
		Hobbies:   []string{"hiking"},
		Location:  &berlin,
	}

	raw, err := json.Marshal(buildQuery(user, Options{RadiusKm: 30, Limit: 5, MinAge: 25, Gender: "female"}))
	if err != nil {
		t.Fatal(err)
	}
	query := string(raw)

	for _, want := range []string{
		`"geo_distance":{"distance":"30km","location":{"lat":52.52,"lon":13.405}}`,
		`"range":{"age":{"gte":25}}`,
		`"term":{"gender":"female"}`,
		`"must_not":[{"ids":{"values":["u1"]}}]`,
		`"gauss":{"location":{"decay":0.5,"offset":"2km","origin":{"lat":52.52,"lon":13.405},"scale":"15km"}}`,
		`"size":5`,
	} {
		if !strings.Contains(query, want) {
			t.Errorf("query does not contain %s\nquery: %s", want, query)
		}
	}

	// duplicate interests with different casing must only be scored once
	if n := strings.Count(query, `"interest":"music"`); n != 1 {
		t.Errorf("expected one clause for interest music, got %d", n)
	}
	if n := strings.Count(query, `"constant_score"`); n != 3 {
		t.Errorf("expected 3 tag clauses, got %d", n)
	}
}

func TestBuildQueryWithoutLocation(t *testing.T) {
	user := &data.User{ID: "u1", Interests: []string{"music"}}

	raw, _ := json.Marshal(buildQuery(user, Options{}.withDefaults()))
	query := string(raw)

	if strings.Contains(query, "geo_distance") || strings.Contains(query, "function_score") {
		t.Errorf("query must not filter by location when the user has none: %s", query)
	}
	if !strings.Contains(query, `"size":10`) {
		t.Errorf("expected default size 10: %s", query)
	}
}

func TestRecommend(t *testing.T) {
	store := &fakeStore{
		users: map[string]*data.User{
			"me": {ID: "me", Interests: []string{"music", "travel"}, Hobbies: []string{"hiking"}, Location: &berlin},
		},
		hits: []search.Hit{
			{Score: 7.5, User: data.User{
				ID: "p1", Name: "Anna", Email: "anna@example.com", Age: 30,
				Interests: []string{"Travel", "art"}, Hobbies: []string{"hiking"},
				Address: data.Address{City: "Potsdam"}, Location: &potsdam,
			}},
			{Score: 1.2, User: data.User{ID: "p2", Name: "Ben"}},
		},
	}

	recs, err := NewService(store).Recommend(context.Background(), "me", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 recommendations, got %d", len(recs))
	}

	anna := recs[0]
	if !reflect.DeepEqual(anna.SharedInterests, []string{"Travel"}) {
		t.Errorf("shared interests = %v", anna.SharedInterests)
	}
	if !reflect.DeepEqual(anna.SharedHobbies, []string{"hiking"}) {
		t.Errorf("shared hobbies = %v", anna.SharedHobbies)
	}
	if anna.DistanceKm == nil || *anna.DistanceKm < 20 || *anna.DistanceKm > 35 {
		t.Errorf("distance Berlin-Potsdam = %v", anna.DistanceKm)
	}
	if anna.City != "Potsdam" || anna.Score != 7.5 {
		t.Errorf("unexpected recommendation %+v", anna)
	}

	if recs[1].DistanceKm != nil {
		t.Errorf("distance must be empty when the candidate has no location")
	}
	if len(recs[1].SharedInterests) != 0 {
		t.Errorf("expected no shared interests, got %v", recs[1].SharedInterests)
	}

	if store.lastRequest["size"] != DefaultLimit {
		t.Errorf("default limit not applied: %v", store.lastRequest["size"])
	}
}

func TestRecommendUnknownUser(t *testing.T) {
	_, err := NewService(&fakeStore{}).Recommend(context.Background(), "nope", Options{})
	if !errors.Is(err, search.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDistanceKm(t *testing.T) {
	got := distanceKm(berlin, hamburg)
	if math.Abs(got-255) > 5 {
		t.Errorf("distance Berlin-Hamburg = %.1f km, want ~255 km", got)
	}
	if distanceKm(berlin, berlin) != 0 {
		t.Errorf("distance to itself must be 0")
	}
}
