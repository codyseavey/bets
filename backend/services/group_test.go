package services

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/codyseavey/bets/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:?_foreign_keys=ON"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.GroupMember{},
		&models.Pool{},
		&models.PoolOption{},
		&models.Bet{},
		&models.PointsLog{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, id, name string) *models.User {
	t.Helper()
	googleID := "google-" + id
	user := &models.User{ID: id, GoogleID: &googleID, Email: id + "@test.com", Name: name}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func TestCreateGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	user := createTestUser(t, db, "user1", "Alice")

	group, err := svc.CreateGroup("Test Group", 500, user.ID)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	if group.Name != "Test Group" {
		t.Errorf("expected name 'Test Group', got '%s'", group.Name)
	}
	if group.DefaultPoints != 500 {
		t.Errorf("expected default_points 500, got %d", group.DefaultPoints)
	}
	if len(group.InviteCode) != 8 {
		t.Errorf("expected 8 char invite code, got '%s' (len %d)", group.InviteCode, len(group.InviteCode))
	}

	// Verify creator is admin with starting points
	var member models.GroupMember
	if err := db.Where("group_id = ? AND user_id = ?", group.ID, user.ID).First(&member).Error; err != nil {
		t.Fatalf("creator not found as member: %v", err)
	}
	if member.Role != "admin" {
		t.Errorf("expected role 'admin', got '%s'", member.Role)
	}
	if member.PointsBalance != 500 {
		t.Errorf("expected 500 starting points, got %d", member.PointsBalance)
	}

	// Verify points log
	var log models.PointsLog
	if err := db.Where("group_id = ? AND user_id = ? AND type = ?", group.ID, user.ID, models.PointsLogInitial).First(&log).Error; err != nil {
		t.Fatalf("initial points log not found: %v", err)
	}
	if log.Amount != 500 {
		t.Errorf("expected log amount 500, got %d", log.Amount)
	}
}

func TestJoinGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	creator := createTestUser(t, db, "creator", "Creator")
	joiner := createTestUser(t, db, "joiner", "Joiner")

	group, err := svc.CreateGroup("Join Test", 1000, creator.ID)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	// Join with invite code
	joined, err := svc.JoinGroup(group.InviteCode, joiner.ID)
	if err != nil {
		t.Fatalf("JoinGroup failed: %v", err)
	}
	if joined.ID != group.ID {
		t.Errorf("joined wrong group")
	}

	// Verify member role and points
	var member models.GroupMember
	if err := db.Where("group_id = ? AND user_id = ?", group.ID, joiner.ID).First(&member).Error; err != nil {
		t.Fatalf("joiner not found as member: %v", err)
	}
	if member.Role != "member" {
		t.Errorf("expected role 'member', got '%s'", member.Role)
	}
	if member.PointsBalance != 1000 {
		t.Errorf("expected 1000 points, got %d", member.PointsBalance)
	}

	// Joining again should be idempotent
	_, err = svc.JoinGroup(group.InviteCode, joiner.ID)
	if err != nil {
		t.Fatalf("re-joining failed: %v", err)
	}

	// Invalid invite code
	_, err = svc.JoinGroup("BADCODE1", joiner.ID)
	if err == nil {
		t.Error("expected error for invalid invite code")
	}
}

func TestGrantPoints(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	admin := createTestUser(t, db, "admin", "Admin")
	member := createTestUser(t, db, "member", "Member")

	group, _ := svc.CreateGroup("Grant Test", 100, admin.ID)
	svc.JoinGroup(group.InviteCode, member.ID)

	// Grant 500 points
	if err := svc.GrantPoints(group.ID, member.ID, 500, "bonus"); err != nil {
		t.Fatalf("GrantPoints failed: %v", err)
	}

	var m models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, member.ID).First(&m)
	if m.PointsBalance != 600 { // 100 initial + 500 granted
		t.Errorf("expected 600 points, got %d", m.PointsBalance)
	}

	// Verify log
	var log models.PointsLog
	db.Where("group_id = ? AND user_id = ? AND type = ?", group.ID, member.ID, models.PointsLogAdminGrant).First(&log)
	if log.Amount != 500 {
		t.Errorf("expected log amount 500, got %d", log.Amount)
	}
}

func TestKickMember(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	admin := createTestUser(t, db, "admin", "Admin")
	member := createTestUser(t, db, "member", "Member")

	group, _ := svc.CreateGroup("Kick Test", 100, admin.ID)
	svc.JoinGroup(group.InviteCode, member.ID)

	if err := svc.KickMember(group.ID, member.ID); err != nil {
		t.Fatalf("KickMember failed: %v", err)
	}

	var count int64
	db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, member.ID).Count(&count)
	if count != 0 {
		t.Error("member should have been removed")
	}
}

func TestGetUserGroups(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	user := createTestUser(t, db, "user1", "User")

	svc.CreateGroup("Group 1", 100, user.ID)
	svc.CreateGroup("Group 2", 200, user.ID)

	groups, err := svc.GetUserGroups(user.ID)
	if err != nil {
		t.Fatalf("GetUserGroups failed: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestDeleteGroup(t *testing.T) {
	db := setupTestDB(t)
	groupSvc := NewGroupService(db)
	poolSvc := NewPoolService(db)

	admin := createTestUser(t, db, "admin", "Admin")
	member := createTestUser(t, db, "member", "Member")

	group, err := groupSvc.CreateGroup("Delete Me", 500, admin.ID)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if _, err := groupSvc.JoinGroup(group.InviteCode, member.ID); err != nil {
		t.Fatalf("JoinGroup failed: %v", err)
	}

	// Create a pool with bets to verify cascade deletion
	pool, err := poolSvc.CreatePool(group.ID, admin.ID, CreatePoolRequest{
		Title:   "Will it blend?",
		Options: []string{"Yes", "No"},
	})
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}
	if _, err := poolSvc.PlaceBet(pool.ID, member.ID, PlaceBetRequest{OptionID: pool.Options[0].ID, Points: 100}); err != nil {
		t.Fatalf("PlaceBet failed: %v", err)
	}

	// Sanity check: data exists before delete
	var poolCount, betCount, memberCount, logCount int64
	db.Model(&models.Pool{}).Where("group_id = ?", group.ID).Count(&poolCount)
	db.Model(&models.Bet{}).Where("pool_id = ?", pool.ID).Count(&betCount)
	db.Model(&models.GroupMember{}).Where("group_id = ?", group.ID).Count(&memberCount)
	db.Model(&models.PointsLog{}).Where("group_id = ?", group.ID).Count(&logCount)

	if poolCount == 0 || betCount == 0 || memberCount == 0 || logCount == 0 {
		t.Fatal("test setup failed: expected data to exist before delete")
	}

	// Delete the group
	if err := groupSvc.DeleteGroup(group.ID); err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

	// Verify everything is gone
	db.Model(&models.Group{}).Where("id = ?", group.ID).Count(&poolCount)
	if poolCount != 0 {
		t.Error("group should be deleted")
	}

	db.Model(&models.Pool{}).Where("group_id = ?", group.ID).Count(&poolCount)
	if poolCount != 0 {
		t.Error("pools should be deleted")
	}

	db.Model(&models.PoolOption{}).Where("pool_id = ?", pool.ID).Count(&poolCount)
	if poolCount != 0 {
		t.Error("pool options should be deleted")
	}

	db.Model(&models.Bet{}).Where("pool_id = ?", pool.ID).Count(&betCount)
	if betCount != 0 {
		t.Error("bets should be deleted")
	}

	db.Model(&models.GroupMember{}).Where("group_id = ?", group.ID).Count(&memberCount)
	if memberCount != 0 {
		t.Error("members should be deleted")
	}

	db.Model(&models.PointsLog{}).Where("group_id = ?", group.ID).Count(&logCount)
	if logCount != 0 {
		t.Error("points logs should be deleted")
	}
}

func TestDeleteGroup_NonexistentGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)

	// Deleting a nonexistent group should not error (no rows affected is fine)
	if err := svc.DeleteGroup("nonexistent-id"); err != nil {
		t.Fatalf("DeleteGroup on nonexistent group should not error, got: %v", err)
	}
}

func TestRegenerateInviteCode(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	user := createTestUser(t, db, "user1", "User")

	group, _ := svc.CreateGroup("Regen Test", 100, user.ID)
	oldCode := group.InviteCode

	newCode, err := svc.RegenerateInviteCode(group.ID)
	if err != nil {
		t.Fatalf("RegenerateInviteCode failed: %v", err)
	}
	if newCode == oldCode {
		t.Error("new code should differ from old code (extremely unlikely collision)")
	}
	if len(newCode) != 8 {
		t.Errorf("expected 8 char code, got %d", len(newCode))
	}
}
