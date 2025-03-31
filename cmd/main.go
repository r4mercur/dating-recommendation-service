package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"recommandation.com/m/routes"
	"recommandation.com/m/search"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	elasticUser := os.Getenv("ELASTIC_USER")
	elasticPassword := os.Getenv("ELASTIC_PASSWORD")

	r := gin.Default()
	search.InitElasticClient(elasticUser, elasticPassword)

	routes.RegisterHealthcheckRoutes(r)
	routes.RegisterUserRoutes(r)

	r.Run()
}
