package models

import "time"

type PoolStatus string

const (
	PoolStatusOpen      PoolStatus = "open"
	PoolStatusLocked    PoolStatus = "locked"
	PoolStatusResolved  PoolStatus = "resolved"
	PoolStatusCancelled PoolStatus = "cancelled"
)

type Pool struct {
	ID          string       `json:"id" gorm:"primaryKey;type:text"`
	GroupID     string       `json:"group_id" gorm:"index;type:text;not null"`
	Title       string       `json:"title" gorm:"type:text;not null"`
	Description string       `json:"description" gorm:"type:text"`
	Status      PoolStatus   `json:"status" gorm:"type:text;not null;default:open"`
	CreatedBy   string       `json:"created_by" gorm:"type:text;not null"`
	ResolvedAt  *time.Time   `json:"resolved_at"`
	CreatedAt   time.Time    `json:"created_at"`
	Creator     User         `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Options     []PoolOption `json:"options,omitempty" gorm:"foreignKey:PoolID"`
	Bets        []Bet        `json:"bets,omitempty" gorm:"foreignKey:PoolID"`
	Group       Group        `json:"-" gorm:"foreignKey:GroupID"`

	// Virtual fields populated by handlers
	WinningOptionID string `json:"winning_option_id,omitempty" gorm:"-"`
	TotalPot        int    `json:"total_pot" gorm:"-"`
	BetCount        int    `json:"bet_count" gorm:"-"`
}

type PoolOption struct {
	ID          string `json:"id" gorm:"primaryKey;type:text"`
	PoolID      string `json:"pool_id" gorm:"index;type:text;not null"`
	Label       string `json:"label" gorm:"type:text;not null"`
	Description string `json:"description" gorm:"type:text"`
}

type Bet struct {
	ID            string     `json:"id" gorm:"primaryKey;type:text"`
	PoolID        string     `json:"pool_id" gorm:"uniqueIndex:idx_pool_user;type:text;not null"`
	UserID        string     `json:"user_id" gorm:"uniqueIndex:idx_pool_user;type:text;not null"`
	OptionID      string     `json:"option_id" gorm:"type:text;not null"`
	PointsWagered int        `json:"points_wagered" gorm:"not null"`
	CreatedAt     time.Time  `json:"created_at"`
	User          User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Option        PoolOption `json:"option,omitempty" gorm:"foreignKey:OptionID"`
}
