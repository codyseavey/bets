package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/codyseavey/bets/middleware"
	"github.com/codyseavey/bets/models"
	"github.com/codyseavey/bets/services"
)

type WebSocketHandler struct {
	hub         *services.Hub
	authService *services.AuthService
	db          *gorm.DB
}

func NewWebSocketHandler(hub *services.Hub, authService *services.AuthService, db *gorm.DB) *WebSocketHandler {
	return &WebSocketHandler{
		hub:         hub,
		authService: authService,
		db:          db,
	}
}

func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	groupID := c.Param("id")

	// Auth via httpOnly cookie (sent automatically on the WS upgrade HTTP request)
	tokenStr, err := c.Cookie(middleware.CookieName)
	if err != nil || tokenStr == "" {
		// Fallback to query param for non-browser clients
		tokenStr = c.Query("token")
	}
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	claims, err := h.authService.ValidateJWT(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Verify group membership
	var member models.GroupMember
	if err := h.db.Where("group_id = ? AND user_id = ?", groupID, claims.UserID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this group"})
		return
	}

	conn := h.hub.Upgrade(c.Writer, c.Request)
	if conn == nil {
		return
	}

	h.hub.AddClient(conn, groupID, claims.UserID)
}
