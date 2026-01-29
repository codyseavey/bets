package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/codyseavey/bets/models"
)

const (
	ContextGroupMember = "group_member"
)

// GroupMemberRequired checks that the authenticated user is a member of the group
// specified by the :id URL param. Sets the GroupMember in context.
func GroupMemberRequired(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		groupID := c.Param("id")

		var member models.GroupMember
		if err := db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not a member of this group"})
			return
		}

		c.Set(ContextGroupMember, &member)
		c.Next()
	}
}

// GroupAdminRequired checks that the authenticated user is an admin of the group.
// Must be used after GroupMemberRequired.
func GroupAdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		member := GetGroupMember(c)
		if member == nil || member.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

func GetGroupMember(c *gin.Context) *models.GroupMember {
	val, exists := c.Get(ContextGroupMember)
	if !exists {
		return nil
	}
	member, ok := val.(*models.GroupMember)
	if !ok {
		return nil
	}
	return member
}
