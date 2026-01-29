package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/codyseavey/bets/middleware"
	"github.com/codyseavey/bets/services"
)

type GroupHandler struct {
	groupService *services.GroupService
	hub          *services.Hub
}

func NewGroupHandler(groupService *services.GroupService, hub *services.Hub) *GroupHandler {
	return &GroupHandler{
		groupService: groupService,
		hub:          hub,
	}
}

type CreateGroupRequest struct {
	Name          string `json:"name" binding:"required"`
	DefaultPoints int    `json:"default_points" binding:"required,gt=0"`
}

func (h *GroupHandler) Create(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	group, err := h.groupService.CreateGroup(req.Name, req.DefaultPoints, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (h *GroupHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	groups, err := h.groupService.GetUserGroups(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *GroupHandler) Get(c *gin.Context) {
	groupID := c.Param("id")
	group, err := h.groupService.GetGroup(groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	c.JSON(http.StatusOK, group)
}

type JoinGroupRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

func (h *GroupHandler) Join(c *gin.Context) {
	var req JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	group, err := h.groupService.JoinGroup(req.InviteCode, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Broadcast member joined event
	user, _ := h.groupService.GetMember(group.ID, userID)
	h.hub.BroadcastToGroup(group.ID, services.WSEvent{
		Type:    "member_joined",
		Payload: user,
	})

	c.JSON(http.StatusOK, group)
}

type UpdateGroupRequest struct {
	Name          string `json:"name" binding:"required"`
	DefaultPoints int    `json:"default_points" binding:"required,gt=0"`
}

func (h *GroupHandler) Update(c *gin.Context) {
	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID := c.Param("id")
	if err := h.groupService.UpdateGroup(groupID, req.Name, req.DefaultPoints); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group updated"})
}

type GrantPointsRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Amount int    `json:"amount" binding:"required"`
	Note   string `json:"note"`
}

func (h *GroupHandler) GrantPoints(c *gin.Context) {
	var req GrantPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID := c.Param("id")
	if err := h.groupService.GrantPoints(groupID, req.UserID, req.Amount, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.hub.BroadcastToGroup(groupID, services.WSEvent{
		Type: "points_granted",
		Payload: gin.H{
			"user_id": req.UserID,
			"amount":  req.Amount,
			"note":    req.Note,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "points granted"})
}

func (h *GroupHandler) KickMember(c *gin.Context) {
	groupID := c.Param("id")
	targetUserID := c.Param("uid")
	userID := middleware.GetUserID(c)

	if targetUserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot kick yourself"})
		return
	}

	if err := h.groupService.KickMember(groupID, targetUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.hub.BroadcastToGroup(groupID, services.WSEvent{
		Type:    "member_kicked",
		Payload: gin.H{"user_id": targetUserID},
	})

	c.JSON(http.StatusOK, gin.H{"message": "member kicked"})
}

func (h *GroupHandler) RegenerateInvite(c *gin.Context) {
	groupID := c.Param("id")
	code, err := h.groupService.RegenerateInviteCode(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"invite_code": code})
}
