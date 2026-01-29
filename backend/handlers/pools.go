package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/codyseavey/bets/middleware"
	"github.com/codyseavey/bets/services"
)

type PoolHandler struct {
	poolService  *services.PoolService
	groupService *services.GroupService
	hub          *services.Hub
}

func NewPoolHandler(poolService *services.PoolService, groupService *services.GroupService, hub *services.Hub) *PoolHandler {
	return &PoolHandler{
		poolService:  poolService,
		groupService: groupService,
		hub:          hub,
	}
}

func (h *PoolHandler) Create(c *gin.Context) {
	var req services.CreatePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID := c.Param("id")
	userID := middleware.GetUserID(c)

	pool, err := h.poolService.CreatePool(groupID, userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.hub.BroadcastToGroup(groupID, services.WSEvent{
		Type:    "pool_created",
		Payload: pool,
	})

	c.JSON(http.StatusCreated, pool)
}

func (h *PoolHandler) List(c *gin.Context) {
	groupID := c.Param("id")
	status := c.Query("status")

	pools, err := h.poolService.GetGroupPools(groupID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pools)
}

func (h *PoolHandler) Get(c *gin.Context) {
	poolID := c.Param("pid")
	pool, err := h.poolService.GetPool(poolID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pool not found"})
		return
	}
	c.JSON(http.StatusOK, pool)
}

func (h *PoolHandler) PlaceBet(c *gin.Context) {
	var req services.PlaceBetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	poolID := c.Param("pid")
	userID := middleware.GetUserID(c)

	bet, err := h.poolService.PlaceBet(poolID, userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get group ID for broadcast
	groupID, _ := h.poolService.GetPoolGroupID(poolID)
	h.hub.BroadcastToGroup(groupID, services.WSEvent{
		Type: "bet_placed",
		Payload: gin.H{
			"pool_id": poolID,
			"user_id": userID,
			"bet_id":  bet.ID,
		},
	})

	c.JSON(http.StatusCreated, bet)
}

func (h *PoolHandler) Lock(c *gin.Context) {
	poolID := c.Param("pid")
	userID := middleware.GetUserID(c)
	member := middleware.GetGroupMember(c)
	isAdmin := member != nil && member.Role == "admin"

	if err := h.poolService.LockPool(poolID, userID, isAdmin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID, _ := h.poolService.GetPoolGroupID(poolID)
	h.hub.BroadcastToGroup(groupID, services.WSEvent{
		Type:    "pool_locked",
		Payload: gin.H{"pool_id": poolID},
	})

	c.JSON(http.StatusOK, gin.H{"message": "pool locked"})
}

type ResolveRequest struct {
	WinningOptionID string `json:"winning_option_id" binding:"required"`
}

func (h *PoolHandler) Resolve(c *gin.Context) {
	var req ResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	poolID := c.Param("pid")
	userID := middleware.GetUserID(c)
	member := middleware.GetGroupMember(c)
	isAdmin := member != nil && member.Role == "admin"

	if err := h.poolService.ResolvePool(poolID, req.WinningOptionID, userID, isAdmin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the updated pool to broadcast full results
	pool, _ := h.poolService.GetPool(poolID)
	groupID, _ := h.poolService.GetPoolGroupID(poolID)
	h.hub.BroadcastToGroup(groupID, services.WSEvent{
		Type:    "pool_resolved",
		Payload: pool,
	})

	c.JSON(http.StatusOK, gin.H{"message": "pool resolved"})
}

func (h *PoolHandler) Cancel(c *gin.Context) {
	poolID := c.Param("pid")
	userID := middleware.GetUserID(c)
	member := middleware.GetGroupMember(c)
	isAdmin := member != nil && member.Role == "admin"

	if err := h.poolService.CancelPool(poolID, userID, isAdmin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID, _ := h.poolService.GetPoolGroupID(poolID)
	h.hub.BroadcastToGroup(groupID, services.WSEvent{
		Type:    "pool_cancelled",
		Payload: gin.H{"pool_id": poolID},
	})

	c.JSON(http.StatusOK, gin.H{"message": "pool cancelled, all bets refunded"})
}
