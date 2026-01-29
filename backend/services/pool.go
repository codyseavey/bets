package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/codyseavey/bets/models"
)

type PoolService struct {
	db *gorm.DB
}

func NewPoolService(db *gorm.DB) *PoolService {
	return &PoolService{db: db}
}

type CreatePoolRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	Options     []string `json:"options" binding:"required,min=2"`
}

func (s *PoolService) CreatePool(groupID, userID string, req CreatePoolRequest) (*models.Pool, error) {
	pool := &models.Pool{
		ID:          uuid.New().String(),
		GroupID:     groupID,
		Title:       req.Title,
		Description: req.Description,
		Status:      models.PoolStatusOpen,
		CreatedBy:   userID,
	}

	tx := s.db.Begin()
	if err := tx.Create(pool).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	for _, label := range req.Options {
		opt := &models.PoolOption{
			ID:     uuid.New().String(),
			PoolID: pool.ID,
			Label:  label,
		}
		if err := tx.Create(opt).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create option: %w", err)
		}
		pool.Options = append(pool.Options, *opt)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return pool, nil
}

func (s *PoolService) GetGroupPools(groupID string, status string) ([]models.Pool, error) {
	query := s.db.Where("group_id = ?", groupID).Preload("Options").Preload("Creator").Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var pools []models.Pool
	if err := query.Find(&pools).Error; err != nil {
		return nil, err
	}

	// Populate virtual fields
	for i := range pools {
		s.populatePoolStats(&pools[i])
	}

	return pools, nil
}

func (s *PoolService) GetPool(poolID string) (*models.Pool, error) {
	var pool models.Pool
	err := s.db.
		Preload("Options").
		Preload("Creator").
		Preload("Bets.User").
		Preload("Bets.Option").
		First(&pool, "id = ?", poolID).Error
	if err != nil {
		return nil, err
	}
	s.populatePoolStats(&pool)
	return &pool, nil
}

type PlaceBetRequest struct {
	OptionID string `json:"option_id" binding:"required"`
	Points   int    `json:"points" binding:"required,gt=0"`
}

func (s *PoolService) PlaceBet(poolID, userID string, req PlaceBetRequest) (*models.Bet, error) {
	tx := s.db.Begin()

	// Get pool and verify it's open
	var pool models.Pool
	if err := tx.First(&pool, "id = ?", poolID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("pool not found")
	}
	if pool.Status != models.PoolStatusOpen {
		tx.Rollback()
		return nil, fmt.Errorf("pool is not open for bets")
	}

	// Verify option belongs to pool
	var option models.PoolOption
	if err := tx.First(&option, "id = ? AND pool_id = ?", req.OptionID, poolID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("invalid option for this pool")
	}

	// Check user hasn't already bet on this pool
	var existingCount int64
	tx.Model(&models.Bet{}).Where("pool_id = ? AND user_id = ?", poolID, userID).Count(&existingCount)
	if existingCount > 0 {
		tx.Rollback()
		return nil, fmt.Errorf("you already placed a bet on this pool")
	}

	// Deduct points from member
	var member models.GroupMember
	if err := tx.Where("group_id = ? AND user_id = ?", pool.GroupID, userID).First(&member).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("not a member of this group")
	}
	if member.PointsBalance < req.Points {
		tx.Rollback()
		return nil, fmt.Errorf("insufficient points (have %d, need %d)", member.PointsBalance, req.Points)
	}

	member.PointsBalance -= req.Points
	if err := tx.Save(&member).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	bet := &models.Bet{
		ID:            uuid.New().String(),
		PoolID:        poolID,
		UserID:        userID,
		OptionID:      req.OptionID,
		PointsWagered: req.Points,
	}
	if err := tx.Create(bet).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to place bet: %w", err)
	}

	logEntry := &models.PointsLog{
		ID:          uuid.New().String(),
		GroupID:     pool.GroupID,
		UserID:      userID,
		Amount:      -req.Points,
		Type:        models.PointsLogBetPlaced,
		ReferenceID: bet.ID,
		Note:        fmt.Sprintf("Bet on \"%s\" in pool \"%s\"", option.Label, pool.Title),
	}
	if err := tx.Create(logEntry).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return bet, nil
}

func (s *PoolService) LockPool(poolID, userID string, isAdmin bool) error {
	var pool models.Pool
	if err := s.db.First(&pool, "id = ?", poolID).Error; err != nil {
		return fmt.Errorf("pool not found")
	}
	if pool.Status != models.PoolStatusOpen {
		return fmt.Errorf("pool is not open")
	}
	if pool.CreatedBy != userID && !isAdmin {
		return fmt.Errorf("only pool creator or group admin can lock")
	}

	return s.db.Model(&pool).Update("status", models.PoolStatusLocked).Error
}

func (s *PoolService) ResolvePool(poolID, winningOptionID, userID string, isAdmin bool) error {
	tx := s.db.Begin()

	var pool models.Pool
	if err := tx.First(&pool, "id = ?", poolID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("pool not found")
	}
	if pool.Status != models.PoolStatusOpen && pool.Status != models.PoolStatusLocked {
		tx.Rollback()
		return fmt.Errorf("pool cannot be resolved (status: %s)", pool.Status)
	}
	if pool.CreatedBy != userID && !isAdmin {
		tx.Rollback()
		return fmt.Errorf("only pool creator or group admin can resolve")
	}

	// Verify winning option
	var option models.PoolOption
	if err := tx.First(&option, "id = ? AND pool_id = ?", winningOptionID, poolID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("invalid winning option")
	}

	// Get all bets
	var bets []models.Bet
	if err := tx.Where("pool_id = ?", poolID).Find(&bets).Error; err != nil {
		tx.Rollback()
		return err
	}

	totalPot := 0
	totalWinningWagers := 0
	for _, b := range bets {
		totalPot += b.PointsWagered
		if b.OptionID == winningOptionID {
			totalWinningWagers += b.PointsWagered
		}
	}

	if totalWinningWagers == 0 {
		// Nobody picked the winner, refund everyone
		for _, b := range bets {
			if err := s.creditMember(tx, pool.GroupID, b.UserID, b.PointsWagered, models.PointsLogBetRefund, b.ID, "No winners, bet refunded"); err != nil {
				tx.Rollback()
				return err
			}
		}
	} else {
		// Distribute pot proportionally to winners
		distributed := 0
		winnerIndex := 0
		winnerCount := 0
		for _, b := range bets {
			if b.OptionID == winningOptionID {
				winnerCount++
			}
		}

		for _, b := range bets {
			if b.OptionID != winningOptionID {
				continue
			}
			winnerIndex++
			var winnings int
			if winnerIndex == winnerCount {
				// Last winner gets remainder to avoid rounding loss
				winnings = totalPot - distributed
			} else {
				winnings = (b.PointsWagered * totalPot) / totalWinningWagers
			}
			distributed += winnings

			if err := s.creditMember(tx, pool.GroupID, b.UserID, winnings, models.PointsLogBetWon, b.ID,
				fmt.Sprintf("Won %d points from pool \"%s\"", winnings, pool.Title)); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	now := time.Now()
	if err := tx.Model(&pool).Updates(map[string]interface{}{
		"status":      models.PoolStatusResolved,
		"resolved_at": now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Store winning option in a simple way: use the pool's description to record it,
	// or better yet, add a winning_option_id column. For now we'll use a PointsLog reference.
	// Actually, let's just store it. We need a place for it. Let's add a simple record.
	// We'll create a PointsLog entry for the resolution event itself.
	resolutionLog := &models.PointsLog{
		ID:          uuid.New().String(),
		GroupID:     pool.GroupID,
		UserID:      userID,
		Amount:      0,
		Type:        "pool_resolved",
		ReferenceID: winningOptionID,
		Note:        fmt.Sprintf("Resolved pool \"%s\" - winning option: \"%s\"", pool.Title, option.Label),
	}
	if err := tx.Create(resolutionLog).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *PoolService) CancelPool(poolID, userID string, isAdmin bool) error {
	tx := s.db.Begin()

	var pool models.Pool
	if err := tx.First(&pool, "id = ?", poolID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("pool not found")
	}
	if pool.Status == models.PoolStatusResolved || pool.Status == models.PoolStatusCancelled {
		tx.Rollback()
		return fmt.Errorf("pool is already %s", pool.Status)
	}
	if pool.CreatedBy != userID && !isAdmin {
		tx.Rollback()
		return fmt.Errorf("only pool creator or group admin can cancel")
	}

	// Refund all bets
	var bets []models.Bet
	if err := tx.Where("pool_id = ?", poolID).Find(&bets).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, b := range bets {
		if err := s.creditMember(tx, pool.GroupID, b.UserID, b.PointsWagered, models.PointsLogBetRefund, b.ID, "Pool cancelled, bet refunded"); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Model(&pool).Update("status", models.PoolStatusCancelled).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *PoolService) creditMember(tx *gorm.DB, groupID, userID string, amount int, logType models.PointsLogType, refID, note string) error {
	result := tx.Model(&models.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("points_balance", gorm.Expr("points_balance + ?", amount))
	if result.Error != nil {
		return result.Error
	}

	logEntry := &models.PointsLog{
		ID:          uuid.New().String(),
		GroupID:     groupID,
		UserID:      userID,
		Amount:      amount,
		Type:        logType,
		ReferenceID: refID,
		Note:        note,
	}
	return tx.Create(logEntry).Error
}

func (s *PoolService) populatePoolStats(pool *models.Pool) {
	var totalPot int64
	var betCount int64
	s.db.Model(&models.Bet{}).Where("pool_id = ?", pool.ID).Count(&betCount)
	s.db.Model(&models.Bet{}).Where("pool_id = ?", pool.ID).Select("COALESCE(SUM(points_wagered), 0)").Scan(&totalPot)
	pool.TotalPot = int(totalPot)
	pool.BetCount = int(betCount)

	// If resolved, find winning option from the resolution log
	if pool.Status == models.PoolStatusResolved {
		var log models.PointsLog
		if err := s.db.Where("type = ? AND group_id = ? AND note LIKE ?",
			"pool_resolved", pool.GroupID, fmt.Sprintf("Resolved pool \"%s\"%%", pool.Title)).
			First(&log).Error; err == nil {
			pool.WinningOptionID = log.ReferenceID
		}
	}
}

func (s *PoolService) GetPoolGroupID(poolID string) (string, error) {
	var pool models.Pool
	if err := s.db.Select("group_id").First(&pool, "id = ?", poolID).Error; err != nil {
		return "", err
	}
	return pool.GroupID, nil
}
