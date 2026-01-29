package models

import "time"

type User struct {
	ID           string    `json:"id" gorm:"primaryKey;type:text"`
	GoogleID     *string   `json:"-" gorm:"uniqueIndex;type:text"`
	Email        string    `json:"email" gorm:"uniqueIndex;type:text;not null"`
	Name         string    `json:"name" gorm:"type:text;not null"`
	AvatarURL    string    `json:"avatar_url" gorm:"type:text"`
	PasswordHash string    `json:"-" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
}
