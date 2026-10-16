package importer

import (
	"math/rand/v2"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
	"recommandation.com/m/data"
)

var (
	interests = []string{"music", "sports", "movies", "books", "travel", "art", "food", "technology", "nature", "fashion"}
	hobbies   = []string{"hiking", "cooking", "gaming", "photography", "running", "yoga", "dancing", "reading", "cycling", "painting"}
)

// Fake users are spread around real cities so that radius searches return realistic results.
var cities = []struct {
	name     string
	location data.GeoPoint
}{
	{"Berlin", data.GeoPoint{Lat: 52.5200, Lon: 13.4050}},
	{"Hamburg", data.GeoPoint{Lat: 53.5511, Lon: 9.9937}},
	{"München", data.GeoPoint{Lat: 48.1351, Lon: 11.5820}},
	{"Köln", data.GeoPoint{Lat: 50.9375, Lon: 6.9603}},
	{"Frankfurt am Main", data.GeoPoint{Lat: 50.1109, Lon: 8.6821}},
	{"Stuttgart", data.GeoPoint{Lat: 48.7758, Lon: 9.1829}},
	{"Düsseldorf", data.GeoPoint{Lat: 51.2277, Lon: 6.7735}},
	{"Leipzig", data.GeoPoint{Lat: 51.3397, Lon: 12.3731}},
}

// maxJitterDeg moves users up to roughly 20 km away from the city center.
const maxJitterDeg = 0.2

func createFakeUsers(count int) []*data.User {
	gofakeit.Seed(0)

	users := make([]*data.User, 0, count)
	for range count {
		city := cities[rand.IntN(len(cities))]
		users = append(users, &data.User{
			ID:        gofakeit.UUID(),
			Name:      gofakeit.Name(),
			Email:     gofakeit.Email(),
			Interests: pick(interests, 2, 4),
			Hobbies:   pick(hobbies, 2, 4),
			Age:       gofakeit.Number(18, 60),
			Address: data.Address{
				Street:  gofakeit.Street(),
				City:    city.name,
				Zip:     gofakeit.Zip(),
				Country: "Germany",
			},
			Location: &data.GeoPoint{
				Lat: city.location.Lat + jitter(),
				Lon: city.location.Lon + jitter(),
			},
			Gender: strings.ToUpper(gofakeit.Gender()),
			Status: "ACTIVE",
			Photo:  "https://example.com/photo.jpg",
		})
	}
	return users
}

// pick returns between minN and maxN distinct random elements of values.
func pick(values []string, minN, maxN int) []string {
	n := minN + rand.IntN(maxN-minN+1)
	picked := make([]string, 0, n)
	for _, idx := range rand.Perm(len(values))[:n] {
		picked = append(picked, values[idx])
	}
	return picked
}

func jitter() float64 {
	return (rand.Float64()*2 - 1) * maxJitterDeg
}
