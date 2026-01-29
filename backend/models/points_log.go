package models

import "time"

type PointsLogType string

const (
	PointsLogInitial    PointsLogType = "initial"
	PointsLogAdminGrant PointsLogType = "admin_grant"
	PointsLogBetPlaced  PointsLogType = "bet_placed"
	PointsLogBetWon     PointsLogType = "bet_won"
	PointsLogBetRefund  PointsLogType = "bet_refund"
)

type PointsLog struct {
	ID          string        `json:"id" gorm:"primaryKey;type:text"`
	GroupID     string        `json:"group_id" gorm:"index;type:text;not null"`
	UserID      string        `json:"user_id" gorm:"index;type:text;not null"`
	Amount      int           `json:"amount" gorm:"not null"`
	Type        PointsLogType `json:"type" gorm:"type:text;not null"`
	ReferenceID string        `json:"reference_id" gorm:"type:text"`
	Note        string        `json:"note" gorm:"type:text"`
	CreatedAt   time.Time     `json:"created_at"`
	User        User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
}
