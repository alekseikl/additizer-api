package models

import (
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

type PresetGroup struct {
	gorm.Model
	UserID uuid.UUID `gorm:"type:uuid;index;not null"`
	User   *User     `gorm:"foreignKey:UserID"`
	Name   string    `gorm:"index;size:255;not null"`
	Public bool      `gorm:"index"`
}

type Preset struct {
	gorm.Model
	GroupId    uint           `gorm:"index;not null"`
	Group      *PresetGroup   `gorm:"foreignKey:GroupId"`
	Type       ModuleType     `gorm:"index;size:255;not null"`
	Name       string         `gorm:"index;size:255;not null"`
	Public     bool           `gorm:"index"`
	AppVersion string         `gorm:"size:255;not null"`
	Preset     datatypes.JSON `gorm:"type:jsonb"`
}

type PresetShare struct {
	gorm.Model
	PresetID uint      `gorm:"not null;uniqueIndex:ux_preset_user,priority:1"`
	Preset   *Preset   `gorm:"foreignKey:PresetID"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:ux_preset_user,priority:2"`
	User     *User     `gorm:"foreignKey:UserID"`
}

type PresetGroupShare struct {
	gorm.Model
	GroupID uint         `gorm:"not null;uniqueIndex:ux_preset_group_user,priority:1"`
	Group   *PresetGroup `gorm:"foreignKey:GroupID"`
	UserID  uuid.UUID    `gorm:"type:uuid;not null;index;uniqueIndex:ux_preset_group_user,priority:2"`
	User    *User        `gorm:"foreignKey:UserID"`
}
