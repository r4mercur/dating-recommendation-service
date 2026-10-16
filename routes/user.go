package routes

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"recommandation.com/m/data"
	"recommandation.com/m/importer"
	"recommandation.com/m/recommendation"
	"recommandation.com/m/search"

	"github.com/gin-gonic/gin"
)

const (
	defaultFakeUserCount = 1000
	maxFakeUserCount     = 100000
)

type UserHandler struct {
	importer    *importer.Importer
	recommender *recommendation.Service
}

func NewUserHandler(importer *importer.Importer, recommender *recommendation.Service) *UserHandler {
	return &UserHandler{importer: importer, recommender: recommender}
}

func RegisterUserRoutes(router *gin.Engine, h *UserHandler) {
	userRoutes := router.Group("/users")
	{
		userRoutes.POST("/import/user", h.importUser)
		userRoutes.POST("/create/fake-users", h.importFakeUsers)
		userRoutes.GET("/recommendations/:userId", h.getUserRecommendations)
	}
}

type recommendationQuery struct {
	RadiusKm float64 `form:"radiusKm" binding:"omitempty,gt=0,lte=20000"`
	Limit    int     `form:"limit" binding:"omitempty,min=1,max=100"`
	MinAge   int     `form:"minAge" binding:"omitempty,min=18"`
	MaxAge   int     `form:"maxAge" binding:"omitempty,min=18"`
	Gender   string  `form:"gender"`
}

func (h *UserHandler) getUserRecommendations(c *gin.Context) {
	userId := c.Param("userId")

	var q recommendationQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if q.MinAge > 0 && q.MaxAge > 0 && q.MinAge > q.MaxAge {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minAge must not be greater than maxAge"})
		return
	}

	recommendations, err := h.recommender.Recommend(c.Request.Context(), userId, recommendation.Options{
		RadiusKm: q.RadiusKm,
		Limit:    q.Limit,
		MinAge:   q.MinAge,
		MaxAge:   q.MaxAge,
		Gender:   q.Gender,
	})
	if errors.Is(err, search.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		log.Printf("Error fetching recommendations for user %s: %s", userId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch recommendations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":          userId,
		"recommendations": recommendations,
	})
}

func (h *UserHandler) importFakeUsers(c *gin.Context) {
	count := defaultFakeUserCount
	if raw := c.Query("count"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxFakeUserCount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "count must be between 1 and 100000"})
			return
		}
		count = n
	}

	err := h.importer.ImportFakeUsers(c.Request.Context(), count)
	if errors.Is(err, importer.ErrIndexNotEmpty) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		log.Printf("Error importing fake users: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Created fake users and added to the index",
		"count":   count,
	})
}

func (h *UserHandler) importUser(c *gin.Context) {
	var user data.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.importer.ImportUser(c.Request.Context(), &user); err != nil {
		log.Printf("Error importing user %s: %s", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User imported"})
}
