package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ModuleType string

const (
	HarmonicsEditor ModuleType = "harmonics-editor"
	Oscillator      ModuleType = "oscillator"
	SpectralFilter  ModuleType = "spectral-filter"
	Envelope        ModuleType = "envelope"
	SpectralMixer   ModuleType = "spectral-mixer"
	SpectralBlend   ModuleType = "spectral-blend"
	Mixer           ModuleType = "mixer"
	Amplifier       ModuleType = "amplifier"
	Waveshaper      ModuleType = "waveshaper"
)

type Preset struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;index;not null" json:"user_id"`
	User      *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Type      ModuleType     `gorm:"index;size:255;not null" json:"type"`
	Name      string         `gorm:"index;size:255;not null" json:"name"`
	Public    bool           `gorm:"index" json:"public"`
	Preset    datatypes.JSON `gorm:"type:jsonb" json:"preset"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
