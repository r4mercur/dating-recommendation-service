package routes

import (
	"net/http"

	"recommandation.com/m/data"
	"recommandation.com/m/importer"
	"recommandation.com/m/search"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(router *gin.Engine) {
	userRoutes := router.Group("/users")
	{
		userRoutes.POST("/import/user", importUser)
		userRoutes.POST("/create/fake-users", importFakeUsers)
		userRoutes.GET("/recommendations/:userId", getUserRecommendations)
	}
}

func getUserRecommendations(c *gin.Context) {
	userId := c.Param("userId")
	recommendations, err := search.FetchRecommendationsFromElastic(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"userId":          userId,
		"recommendations": recommendations,
	})
}

func importFakeUsers(c *gin.Context) {
	importer.ImportUsersAndAddToElasticIndex()
	c.JSON(http.StatusOK, gin.H{
		"message": "Created fake users and added to the index",
	})
}

func importUser(c *gin.Context) {
	var user data.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}

	if err := importer.ImportSingleUserAndAddToElasticIndex(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User imported"})
}
