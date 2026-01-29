package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/codyseavey/bets/models"
)

type LeaderboardHandler struct {
	db *gorm.DB
}

func NewLeaderboardHandler(db *gorm.DB) *LeaderboardHandler {
	return &LeaderboardHandler{db: db}
}

type LeaderboardEntry struct {
	UserID        string `json:"user_id"`
	Name          string `json:"name"`
	AvatarURL     string `json:"avatar_url"`
	PointsBalance int    `json:"points_balance"`
	TotalWins     int64  `json:"total_wins"`
	TotalLosses   int64  `json:"total_losses"`
	TotalBets     int64  `json:"total_bets"`
	Rank          int    `json:"rank"`
}

func (h *LeaderboardHandler) GetLeaderboard(c *gin.Context) {
	groupID := c.Param("id")

	var members []models.GroupMember
	if err := h.db.Where("group_id = ?", groupID).
		Preload("User").
		Order("points_balance DESC").
		Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entries := make([]LeaderboardEntry, 0, len(members))
	for i, m := range members {
		var totalBets int64
		h.db.Model(&models.Bet{}).
			Joins("JOIN pools ON pools.id = bets.pool_id").
			Where("bets.user_id = ? AND pools.group_id = ?", m.UserID, groupID).
			Count(&totalBets)

		var totalWins int64
		h.db.Model(&models.PointsLog{}).
			Where("user_id = ? AND group_id = ? AND type = ?", m.UserID, groupID, models.PointsLogBetWon).
			Count(&totalWins)

		// Losses = bets placed on pools that resolved, where user didn't win
		var totalLosses int64
		h.db.Model(&models.PointsLog{}).
			Where("user_id = ? AND group_id = ? AND type = ?", m.UserID, groupID, models.PointsLogBetPlaced).
			Count(&totalLosses)
		// Subtract wins and refunds to get actual losses
		var refunds int64
		h.db.Model(&models.PointsLog{}).
			Where("user_id = ? AND group_id = ? AND type = ?", m.UserID, groupID, models.PointsLogBetRefund).
			Count(&refunds)
		actualLosses := totalLosses - totalWins - refunds
		if actualLosses < 0 {
			actualLosses = 0
		}

		entries = append(entries, LeaderboardEntry{
			UserID:        m.UserID,
			Name:          m.User.Name,
			AvatarURL:     m.User.AvatarURL,
			PointsBalance: m.PointsBalance,
			TotalWins:     totalWins,
			TotalLosses:   actualLosses,
			TotalBets:     totalBets,
			Rank:          i + 1,
		})
	}

	c.JSON(http.StatusOK, entries)
}

func (h *LeaderboardHandler) GetHistory(c *gin.Context) {
	groupID := c.Param("id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var logs []models.PointsLog
	var total int64

	h.db.Model(&models.PointsLog{}).Where("group_id = ?", groupID).Count(&total)
	if err := h.db.Where("group_id = ?", groupID).
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

type GroupStats struct {
	TotalPools               int64 `json:"total_pools"`
	OpenPools                int64 `json:"open_pools"`
	ResolvedPools            int64 `json:"resolved_pools"`
	TotalBets                int64 `json:"total_bets"`
	TotalMembers             int64 `json:"total_members"`
	TotalPointsInCirculation int64 `json:"total_points_in_circulation"`
}

func (h *LeaderboardHandler) GetStats(c *gin.Context) {
	groupID := c.Param("id")

	var stats GroupStats
	h.db.Model(&models.Pool{}).Where("group_id = ?", groupID).Count(&stats.TotalPools)
	h.db.Model(&models.Pool{}).Where("group_id = ? AND status = ?", groupID, models.PoolStatusOpen).Count(&stats.OpenPools)
	h.db.Model(&models.Pool{}).Where("group_id = ? AND status = ?", groupID, models.PoolStatusResolved).Count(&stats.ResolvedPools)
	h.db.Model(&models.Bet{}).Joins("JOIN pools ON pools.id = bets.pool_id").Where("pools.group_id = ?", groupID).Count(&stats.TotalBets)
	h.db.Model(&models.GroupMember{}).Where("group_id = ?", groupID).Count(&stats.TotalMembers)

	var totalPoints int64
	h.db.Model(&models.GroupMember{}).Where("group_id = ?", groupID).Select("COALESCE(SUM(points_balance), 0)").Scan(&totalPoints)
	stats.TotalPointsInCirculation = totalPoints

	c.JSON(http.StatusOK, stats)
}
