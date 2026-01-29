package models

import "time"

type Group struct {
	ID            string        `json:"id" gorm:"primaryKey;type:text"`
	Name          string        `json:"name" gorm:"type:text;not null"`
	InviteCode    string        `json:"invite_code" gorm:"uniqueIndex;type:text;not null"`
	DefaultPoints int           `json:"default_points" gorm:"not null;default:1000"`
	CreatedBy     string        `json:"created_by" gorm:"type:text;not null"`
	Creator       User          `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Members       []GroupMember `json:"members,omitempty" gorm:"foreignKey:GroupID"`
	CreatedAt     time.Time     `json:"created_at"`
}

type GroupMember struct {
	GroupID       string    `json:"group_id" gorm:"primaryKey;type:text"`
	UserID        string    `json:"user_id" gorm:"primaryKey;type:text"`
	Role          string    `json:"role" gorm:"type:text;not null;default:member"` // "admin" or "member"
	PointsBalance int       `json:"points_balance" gorm:"not null;default:0"`
	JoinedAt      time.Time `json:"joined_at"`
	User          User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Group         Group     `json:"-" gorm:"foreignKey:GroupID"`
}
