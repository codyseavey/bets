package services

import (
	"testing"

	"gorm.io/gorm"

	"github.com/codyseavey/bets/models"
)

func setupPoolTest(t *testing.T) (*gorm.DB, *PoolService, *GroupService, *models.Group, *models.User, *models.User) {
	t.Helper()
	db := setupTestDB(t)
	poolSvc := NewPoolService(db)
	groupSvc := NewGroupService(db)

	alice := createTestUser(t, db, "alice", "Alice")
	bob := createTestUser(t, db, "bob", "Bob")

	group, err := groupSvc.CreateGroup("Pool Test Group", 1000, alice.ID)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if _, err := groupSvc.JoinGroup(group.InviteCode, bob.ID); err != nil {
		t.Fatalf("JoinGroup failed: %v", err)
	}

	return db, poolSvc, groupSvc, group, alice, bob
}

func TestCreatePool(t *testing.T) {
	_, poolSvc, _, group, alice, _ := setupPoolTest(t)

	pool, err := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "Who wins?",
		Options: []string{"Team A", "Team B"},
	})
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	if pool.Title != "Who wins?" {
		t.Errorf("expected title 'Who wins?', got '%s'", pool.Title)
	}
	if pool.Status != models.PoolStatusOpen {
		t.Errorf("expected status 'open', got '%s'", pool.Status)
	}
	if len(pool.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(pool.Options))
	}
}

func TestPlaceBet(t *testing.T) {
	db, poolSvc, _, group, alice, bob := setupPoolTest(t)

	pool, _ := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "Test Bet",
		Options: []string{"Yes", "No"},
	})

	optionA := pool.Options[0]

	// Alice bets 200 on option A
	bet, err := poolSvc.PlaceBet(pool.ID, alice.ID, PlaceBetRequest{
		OptionID: optionA.ID,
		Points:   200,
	})
	if err != nil {
		t.Fatalf("PlaceBet failed: %v", err)
	}
	if bet.PointsWagered != 200 {
		t.Errorf("expected 200 wagered, got %d", bet.PointsWagered)
	}

	// Verify points deducted
	var member models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, alice.ID).First(&member)
	if member.PointsBalance != 800 {
		t.Errorf("expected 800 points after bet, got %d", member.PointsBalance)
	}

	// Alice can't bet twice
	_, err = poolSvc.PlaceBet(pool.ID, alice.ID, PlaceBetRequest{
		OptionID: optionA.ID,
		Points:   100,
	})
	if err == nil {
		t.Error("expected error for duplicate bet")
	}

	// Bob can bet
	_, err = poolSvc.PlaceBet(pool.ID, bob.ID, PlaceBetRequest{
		OptionID: pool.Options[1].ID,
		Points:   300,
	})
	if err != nil {
		t.Fatalf("Bob's bet failed: %v", err)
	}
}

func TestInsufficientPoints(t *testing.T) {
	_, poolSvc, _, group, alice, _ := setupPoolTest(t)

	pool, _ := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "Big Bet",
		Options: []string{"A", "B"},
	})

	// Try to bet more than balance
	_, err := poolSvc.PlaceBet(pool.ID, alice.ID, PlaceBetRequest{
		OptionID: pool.Options[0].ID,
		Points:   9999,
	})
	if err == nil {
		t.Error("expected insufficient points error")
	}
}

func TestLockPool(t *testing.T) {
	_, poolSvc, _, group, alice, bob := setupPoolTest(t)

	pool, _ := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "Lock Test",
		Options: []string{"A", "B"},
	})

	// Non-creator, non-admin can't lock
	err := poolSvc.LockPool(pool.ID, bob.ID, false)
	if err == nil {
		t.Error("expected error for non-creator lock")
	}

	// Creator can lock
	if err := poolSvc.LockPool(pool.ID, alice.ID, false); err != nil {
		t.Fatalf("LockPool failed: %v", err)
	}

	// Can't bet on locked pool
	_, err = poolSvc.PlaceBet(pool.ID, bob.ID, PlaceBetRequest{
		OptionID: pool.Options[0].ID,
		Points:   100,
	})
	if err == nil {
		t.Error("expected error betting on locked pool")
	}
}

func TestResolvePool_WinnersGetPot(t *testing.T) {
	db, poolSvc, _, group, alice, bob := setupPoolTest(t)

	pool, _ := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "Resolve Test",
		Options: []string{"Winner", "Loser"},
	})
	winnerOpt := pool.Options[0]
	loserOpt := pool.Options[1]

	// Alice bets 200 on winner
	poolSvc.PlaceBet(pool.ID, alice.ID, PlaceBetRequest{OptionID: winnerOpt.ID, Points: 200})
	// Bob bets 300 on loser
	poolSvc.PlaceBet(pool.ID, bob.ID, PlaceBetRequest{OptionID: loserOpt.ID, Points: 300})

	// Alice: 1000 - 200 = 800, Bob: 1000 - 300 = 700

	if err := poolSvc.ResolvePool(pool.ID, winnerOpt.ID, alice.ID, false); err != nil {
		t.Fatalf("ResolvePool failed: %v", err)
	}

	// Alice was only winner, gets entire pot (200 + 300 = 500)
	var aliceMember models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, alice.ID).First(&aliceMember)
	if aliceMember.PointsBalance != 1300 { // 800 + 500 pot
		t.Errorf("expected Alice to have 1300 pts, got %d", aliceMember.PointsBalance)
	}

	// Bob gets nothing
	var bobMember models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, bob.ID).First(&bobMember)
	if bobMember.PointsBalance != 700 {
		t.Errorf("expected Bob to have 700 pts, got %d", bobMember.PointsBalance)
	}
}

func TestResolvePool_ProportionalSplit(t *testing.T) {
	db, poolSvc, groupSvc, group, alice, bob := setupPoolTest(t)

	charlie := createTestUser(t, db, "charlie", "Charlie")
	groupSvc.JoinGroup(group.InviteCode, charlie.ID)

	pool, _ := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "Split Test",
		Options: []string{"Winner", "Loser"},
	})
	winnerOpt := pool.Options[0]
	loserOpt := pool.Options[1]

	// Alice bets 100 on winner
	poolSvc.PlaceBet(pool.ID, alice.ID, PlaceBetRequest{OptionID: winnerOpt.ID, Points: 100})
	// Bob bets 300 on winner
	poolSvc.PlaceBet(pool.ID, bob.ID, PlaceBetRequest{OptionID: winnerOpt.ID, Points: 300})
	// Charlie bets 200 on loser
	poolSvc.PlaceBet(pool.ID, charlie.ID, PlaceBetRequest{OptionID: loserOpt.ID, Points: 200})

	// Total pot = 600, winning wagers = 400
	// Alice gets 100/400 * 600 = 150
	// Bob gets 300/400 * 600 = 450

	if err := poolSvc.ResolvePool(pool.ID, winnerOpt.ID, alice.ID, true); err != nil {
		t.Fatalf("ResolvePool failed: %v", err)
	}

	var aliceMember models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, alice.ID).First(&aliceMember)
	// Alice: 1000 - 100 + 150 = 1050
	if aliceMember.PointsBalance != 1050 {
		t.Errorf("expected Alice 1050, got %d", aliceMember.PointsBalance)
	}

	var bobMember models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, bob.ID).First(&bobMember)
	// Bob: 1000 - 300 + 450 = 1150
	if bobMember.PointsBalance != 1150 {
		t.Errorf("expected Bob 1150, got %d", bobMember.PointsBalance)
	}

	var charlieMember models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, charlie.ID).First(&charlieMember)
	// Charlie: 1000 - 200 = 800
	if charlieMember.PointsBalance != 800 {
		t.Errorf("expected Charlie 800, got %d", charlieMember.PointsBalance)
	}
}

func TestResolvePool_NoWinners_RefundAll(t *testing.T) {
	db, poolSvc, _, group, alice, bob := setupPoolTest(t)

	pool, _ := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "No Winner Test",
		Options: []string{"A", "B", "C"},
	})

	// Everyone bets on A and B, but C wins
	poolSvc.PlaceBet(pool.ID, alice.ID, PlaceBetRequest{OptionID: pool.Options[0].ID, Points: 200})
	poolSvc.PlaceBet(pool.ID, bob.ID, PlaceBetRequest{OptionID: pool.Options[1].ID, Points: 300})

	if err := poolSvc.ResolvePool(pool.ID, pool.Options[2].ID, alice.ID, true); err != nil {
		t.Fatalf("ResolvePool failed: %v", err)
	}

	// Everyone gets refunded
	var aliceMember models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, alice.ID).First(&aliceMember)
	if aliceMember.PointsBalance != 1000 {
		t.Errorf("expected Alice refund to 1000, got %d", aliceMember.PointsBalance)
	}

	var bobMember models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, bob.ID).First(&bobMember)
	if bobMember.PointsBalance != 1000 {
		t.Errorf("expected Bob refund to 1000, got %d", bobMember.PointsBalance)
	}
}

func TestCancelPool_RefundsAllBets(t *testing.T) {
	db, poolSvc, _, group, alice, bob := setupPoolTest(t)

	pool, _ := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "Cancel Test",
		Options: []string{"A", "B"},
	})

	poolSvc.PlaceBet(pool.ID, alice.ID, PlaceBetRequest{OptionID: pool.Options[0].ID, Points: 400})
	poolSvc.PlaceBet(pool.ID, bob.ID, PlaceBetRequest{OptionID: pool.Options[1].ID, Points: 500})

	if err := poolSvc.CancelPool(pool.ID, alice.ID, false); err != nil {
		t.Fatalf("CancelPool failed: %v", err)
	}

	var aliceMember models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, alice.ID).First(&aliceMember)
	if aliceMember.PointsBalance != 1000 {
		t.Errorf("expected Alice refund to 1000, got %d", aliceMember.PointsBalance)
	}

	var bobMember models.GroupMember
	db.Where("group_id = ? AND user_id = ?", group.ID, bob.ID).First(&bobMember)
	if bobMember.PointsBalance != 1000 {
		t.Errorf("expected Bob refund to 1000, got %d", bobMember.PointsBalance)
	}

	// Pool status should be cancelled
	var updatedPool models.Pool
	db.First(&updatedPool, "id = ?", pool.ID)
	if updatedPool.Status != models.PoolStatusCancelled {
		t.Errorf("expected status 'cancelled', got '%s'", updatedPool.Status)
	}
}

func TestCancelPool_AlreadyResolved(t *testing.T) {
	_, poolSvc, _, group, alice, _ := setupPoolTest(t)

	pool, _ := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "Double Cancel Test",
		Options: []string{"A", "B"},
	})

	poolSvc.ResolvePool(pool.ID, pool.Options[0].ID, alice.ID, true)

	err := poolSvc.CancelPool(pool.ID, alice.ID, true)
	if err == nil {
		t.Error("expected error cancelling resolved pool")
	}
}

func TestBetOnClosedPool(t *testing.T) {
	_, poolSvc, _, group, alice, bob := setupPoolTest(t)

	pool, _ := poolSvc.CreatePool(group.ID, alice.ID, CreatePoolRequest{
		Title:   "Closed Pool",
		Options: []string{"A", "B"},
	})

	// Resolve immediately
	poolSvc.ResolvePool(pool.ID, pool.Options[0].ID, alice.ID, true)

	_, err := poolSvc.PlaceBet(pool.ID, bob.ID, PlaceBetRequest{
		OptionID: pool.Options[0].ID,
		Points:   100,
	})
	if err == nil {
		t.Error("expected error betting on resolved pool")
	}
}
