package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/codyseavey/bets/models"
)

type GroupService struct {
	db *gorm.DB
}

func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{db: db}
}

func (s *GroupService) CreateGroup(name string, defaultPoints int, userID string) (*models.Group, error) {
	inviteCode, err := generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", err)
	}

	group := &models.Group{
		ID:            uuid.New().String(),
		Name:          name,
		InviteCode:    inviteCode,
		DefaultPoints: defaultPoints,
		CreatedBy:     userID,
	}

	tx := s.db.Begin()
	if err := tx.Create(group).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	member := &models.GroupMember{
		GroupID:       group.ID,
		UserID:        userID,
		Role:          "admin",
		PointsBalance: defaultPoints,
		JoinedAt:      time.Now(),
	}
	if err := tx.Create(member).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to add creator as member: %w", err)
	}

	pointsLog := &models.PointsLog{
		ID:      uuid.New().String(),
		GroupID: group.ID,
		UserID:  userID,
		Amount:  defaultPoints,
		Type:    models.PointsLogInitial,
		Note:    "Initial points on group creation",
	}
	if err := tx.Create(pointsLog).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to log initial points: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return group, nil
}

func (s *GroupService) GetUserGroups(userID string) ([]models.Group, error) {
	var groups []models.Group
	err := s.db.
		Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ?", userID).
		Preload("Creator").
		Find(&groups).Error
	return groups, err
}

func (s *GroupService) GetGroup(groupID string) (*models.Group, error) {
	var group models.Group
	err := s.db.
		Preload("Creator").
		Preload("Members.User").
		First(&group, "id = ?", groupID).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *GroupService) JoinGroup(inviteCode, userID string) (*models.Group, error) {
	var group models.Group
	if err := s.db.Where("invite_code = ?", inviteCode).First(&group).Error; err != nil {
		return nil, fmt.Errorf("invalid invite code")
	}

	// Check if already a member
	var count int64
	s.db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, userID).Count(&count)
	if count > 0 {
		return &group, nil // already a member, just return the group
	}

	tx := s.db.Begin()

	member := &models.GroupMember{
		GroupID:       group.ID,
		UserID:        userID,
		Role:          "member",
		PointsBalance: group.DefaultPoints,
		JoinedAt:      time.Now(),
	}
	if err := tx.Create(member).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to join group: %w", err)
	}

	pointsLog := &models.PointsLog{
		ID:      uuid.New().String(),
		GroupID: group.ID,
		UserID:  userID,
		Amount:  group.DefaultPoints,
		Type:    models.PointsLogInitial,
		Note:    "Initial points on joining group",
	}
	if err := tx.Create(pointsLog).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to log initial points: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &group, nil
}

func (s *GroupService) UpdateGroup(groupID, name string, defaultPoints int) error {
	return s.db.Model(&models.Group{}).Where("id = ?", groupID).Updates(map[string]interface{}{
		"name":           name,
		"default_points": defaultPoints,
	}).Error
}

func (s *GroupService) GrantPoints(groupID, targetUserID string, amount int, note string) error {
	tx := s.db.Begin()

	result := tx.Model(&models.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, targetUserID).
		Update("points_balance", gorm.Expr("points_balance + ?", amount))
	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("member not found")
	}

	logEntry := &models.PointsLog{
		ID:      uuid.New().String(),
		GroupID: groupID,
		UserID:  targetUserID,
		Amount:  amount,
		Type:    models.PointsLogAdminGrant,
		Note:    note,
	}
	if err := tx.Create(logEntry).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *GroupService) KickMember(groupID, targetUserID string) error {
	return s.db.Where("group_id = ? AND user_id = ?", groupID, targetUserID).Delete(&models.GroupMember{}).Error
}

func (s *GroupService) RegenerateInviteCode(groupID string) (string, error) {
	code, err := generateInviteCode()
	if err != nil {
		return "", err
	}
	if err := s.db.Model(&models.Group{}).Where("id = ?", groupID).Update("invite_code", code).Error; err != nil {
		return "", err
	}
	return code, nil
}

func (s *GroupService) DeleteGroup(groupID string) error {
	tx := s.db.Begin()

	// Delete in dependency order: bets -> pool options -> pools -> points logs -> members -> group
	// First get all pool IDs for this group
	var poolIDs []string
	if err := tx.Model(&models.Pool{}).Where("group_id = ?", groupID).Pluck("id", &poolIDs).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find pools: %w", err)
	}

	if len(poolIDs) > 0 {
		if err := tx.Where("pool_id IN ?", poolIDs).Delete(&models.Bet{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete bets: %w", err)
		}
		if err := tx.Where("pool_id IN ?", poolIDs).Delete(&models.PoolOption{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete pool options: %w", err)
		}
	}

	if err := tx.Where("group_id = ?", groupID).Delete(&models.Pool{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete pools: %w", err)
	}
	if err := tx.Where("group_id = ?", groupID).Delete(&models.PointsLog{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete points logs: %w", err)
	}
	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMember{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete members: %w", err)
	}
	if err := tx.Delete(&models.Group{}, "id = ?", groupID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return tx.Commit().Error
}

func (s *GroupService) GetMember(groupID, userID string) (*models.GroupMember, error) {
	var member models.GroupMember
	err := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).Preload("User").First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

const inviteCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars (0/O, 1/I)

func generateInviteCode() (string, error) {
	code := make([]byte, 8)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
		if err != nil {
			return "", err
		}
		code[i] = inviteCodeChars[n.Int64()]
	}
	return string(code), nil
}
