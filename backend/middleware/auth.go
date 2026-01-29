package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/codyseavey/bets/services"
)

const (
	CookieName    = "bets_token"
	ContextUserID = "user_id"
	ContextEmail  = "email"
	ContextName   = "name"
)

func AuthRequired(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie(CookieName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		claims, err := authService.ValidateJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextName, claims.Name)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	return c.GetString(ContextUserID)
}
