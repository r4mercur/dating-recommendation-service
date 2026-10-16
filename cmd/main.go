package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"recommandation.com/m/config"
	"recommandation.com/m/importer"
	"recommandation.com/m/recommendation"
	"recommandation.com/m/routes"
	"recommandation.com/m/search"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables only")
	}
	cfg := config.Load()

	esClient, err := search.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = search.EnsureUserIndex(ctx, esClient, cfg.ElasticIndex)
	cancel()
	if err != nil {
		log.Fatal(err)
	}

	store := search.NewUserStore(esClient, cfg.ElasticIndex)

	var userService *importer.UserServiceClient
	if cfg.UserServiceURL != "" {
		userService = importer.NewUserServiceClient(cfg.UserServiceURL)
	}

	userHandler := routes.NewUserHandler(
		importer.New(store, userService),
		recommendation.NewService(store),
	)

	r := gin.Default()
	routes.RegisterHealthcheckRoutes(r)
	routes.RegisterUserRoutes(r, userHandler)

	log.Fatal(r.Run(":" + cfg.Port))
}
