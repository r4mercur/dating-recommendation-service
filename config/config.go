package config

import "os"

type Config struct {
	Port                      string
	ElasticURL                string
	ElasticUser               string
	ElasticPassword           string
	ElasticIndex              string
	ElasticInsecureSkipVerify bool
	// UserServiceURL is the base URL of the user service that fake users are mirrored to.
	// An empty value disables the mirroring.
	UserServiceURL string
}

func Load() Config {
	return Config{
		Port:                      getEnv("PORT", "3000"),
		ElasticURL:                getEnv("ELASTIC_URL", "https://localhost:9200"),
		ElasticUser:               os.Getenv("ELASTIC_USER"),
		ElasticPassword:           os.Getenv("ELASTIC_PASSWORD"),
		ElasticIndex:              getEnv("ELASTIC_INDEX", "users"),
		ElasticInsecureSkipVerify: getEnv("ELASTIC_INSECURE_SKIP_VERIFY", "true") == "true",
		UserServiceURL:            getEnv("USER_SERVICE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
