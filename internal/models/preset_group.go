package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PresetGroup struct {
	gorm.Model
	UserID uuid.UUID `gorm:"type:uuid;index;not null"`
	User   *User     `gorm:"foreignKey:UserID"`
	Name   string    `gorm:"index;size:255;not null"`
	Public bool      `gorm:"index"`
}
