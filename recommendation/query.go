package recommendation

import (
	"fmt"
	"strings"

	"recommandation.com/m/data"
)

// Score contributions. Every shared interest or hobby adds its weight; proximity adds up
// to distanceWeight, decaying to half of it at distanceOffsetKm + radius/2.
const (
	interestWeight   = 2.0
	hobbyWeight      = 1.0
	distanceWeight   = 3.0
	distanceOffsetKm = 2.0
)

// buildQuery returns a search request that keeps candidates within the radius and ranks
// them by shared interests, shared hobbies and distance. Candidates without any overlap
// are still returned (ranked by distance), so a user always gets suggestions nearby.
func buildQuery(user *data.User, opts Options) map[string]any {
	filter := []any{}
	if user.Location != nil {
		filter = append(filter, map[string]any{
			"geo_distance": map[string]any{
				"distance": km(opts.RadiusKm),
				"location": user.Location,
			},
		})
	}
	if opts.MinAge > 0 || opts.MaxAge > 0 {
		ageRange := map[string]any{}
		if opts.MinAge > 0 {
			ageRange["gte"] = opts.MinAge
		}
		if opts.MaxAge > 0 {
			ageRange["lte"] = opts.MaxAge
		}
		filter = append(filter, map[string]any{"range": map[string]any{"age": ageRange}})
	}
	if opts.Gender != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"gender": opts.Gender}})
	}

	should := []any{}
	should = append(should, tagClauses("interest", user.Interests, interestWeight)...)
	should = append(should, tagClauses("hobby", user.Hobbies, hobbyWeight)...)

	query := map[string]any{
		"bool": map[string]any{
			"filter":   filter,
			"should":   should,
			"must_not": []any{map[string]any{"ids": map[string]any{"values": []string{user.ID}}}},
		},
	}

	if user.Location != nil {
		query = map[string]any{
			"function_score": map[string]any{
				"query": query,
				"functions": []any{
					map[string]any{
						"gauss": map[string]any{
							"location": map[string]any{
								"origin": user.Location,
								"offset": km(distanceOffsetKm),
								"scale":  km(opts.RadiusKm / 2),
								"decay":  0.5,
							},
						},
						"weight": distanceWeight,
					},
				},
				"score_mode": "sum",
				"boost_mode": "sum",
			},
		}
	}

	return map[string]any{
		"query": query,
		"size":  opts.Limit,
	}
}

// tagClauses creates one constant-score clause per distinct tag, so the score grows
// with the number of shared tags instead of their term frequency.
func tagClauses(field string, tags []string, weight float64) []any {
	clauses := []any{}
	seen := map[string]bool{}
	for _, tag := range tags {
		normalized := strings.ToLower(strings.TrimSpace(tag))
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		clauses = append(clauses, map[string]any{
			"constant_score": map[string]any{
				"filter": map[string]any{"term": map[string]any{field: normalized}},
				"boost":  weight,
			},
		})
	}
	return clauses
}

func km(value float64) string {
	return fmt.Sprintf("%gkm", value)
}
