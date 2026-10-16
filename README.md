# User Recommendation Service

A microservice that suggests dating partners to a user based on **location**, shared **interests** and shared **hobbies**.

## How matching works

For `GET /users/recommendations/{userId}` the service loads the user from Elasticsearch and searches for candidates:

1. **Filter**
   - The user is never suggested to themselves.
   - If the user has a `location`: only candidates within `radiusKm` (default 50 km).
   - Optional: age range (`minAge`, `maxAge`) and `gender`.
2. **Score**
   - +2 per shared interest
   - +1 per shared hobby
   - up to +3 for proximity (Gaussian decay: full score within 2 km, half score at 2 km + radius/2)

Candidates without shared interests or hobbies are still returned when they are nearby, so users always get suggestions. Tags are compared case-insensitively.

The weights live in [recommendation/query.go](recommendation/query.go).

## Configuration

Environment variables (a `.env` file is loaded if present):

| Variable                       | Default                  | Description                                           |
|--------------------------------|--------------------------|-------------------------------------------------------|
| `PORT`                         | `3000`                   | HTTP port                                             |
| `ELASTIC_URL`                  | `https://localhost:9200` | Elasticsearch address                                 |
| `ELASTIC_USER`                 |                          | Elasticsearch user                                    |
| `ELASTIC_PASSWORD`             |                          | Elasticsearch password                                |
| `ELASTIC_INDEX`                | `users`                  | Index for user documents                              |
| `ELASTIC_INSECURE_SKIP_VERIFY` | `true`                   | Skip TLS verification (self-signed local certs)       |
| `USER_SERVICE_URL`             | `http://localhost:8080`  | User service that fake users are mirrored to; empty disables it |

On startup the service creates the index with its mapping (`location` as `geo_point`, tags as lowercase keywords).
If an index from an older version without that mapping exists, the service refuses to start; delete it (`DELETE /users`) or choose a new `ELASTIC_INDEX`.

## Running

```bash
docker compose up -d   # Elasticsearch + Kibana
go run ./cmd
```

## API

### Import or update a user

```bash
curl -X POST http://localhost:3000/users/import/user \
  -H "Content-Type: application/json" \
  -d '{
    "id": "2aec576c-6449-4052-92bf-ad7b6fe5487d",
    "name": "Max Mustermann",
    "interest": ["music", "travel"],
    "hobby": ["hiking"],
    "age": 29,
    "gender": "MALE",
    "address": { "city": "Berlin", "country": "Germany" },
    "location": { "lat": 52.517, "lon": 13.389 }
  }'
```

`id` is required; without `location` the user is matched by interests and hobbies only.

### Generate fake users

```bash
curl -X POST "http://localhost:3000/users/create/fake-users?count=1000"
```

Only works on an empty index (otherwise `409 Conflict`). Fake users are spread around several German cities.

### Get partner suggestions

```bash
curl "http://localhost:3000/users/recommendations/{userId}?radiusKm=25&limit=20&minAge=25&maxAge=35&gender=female"
```

| Query parameter | Default | Description               |
|-----------------|---------|---------------------------|
| `radiusKm`      | `50`    | Search radius             |
| `limit`         | `10`    | Number of results (1–100) |
| `minAge`        |         | Minimum age               |
| `maxAge`        |         | Maximum age               |
| `gender`        |         | Gender of the candidates  |

Response:

```json
{
  "userId": "2aec576c-6449-4052-92bf-ad7b6fe5487d",
  "recommendations": [
    {
      "id": "8f1c...",
      "name": "Anna Schmidt",
      "age": 31,
      "gender": "FEMALE",
      "city": "Berlin",
      "photo": "https://example.com/photo.jpg",
      "distanceKm": 4.2,
      "sharedInterests": ["music"],
      "sharedHobbies": ["hiking"],
      "score": 7.8
    }
  ]
}
```

Returns `404` if the user is not in the index.

## Tests

```bash
go test ./...
```
