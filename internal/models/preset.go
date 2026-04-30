package models

import (
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
	gorm.Model
	GroupId    uint           `gorm:"index;not null"`
	Group      *PresetGroup   `gorm:"foreignKey:GroupId"`
	Type       ModuleType     `gorm:"index;size:255;not null"`
	Name       string         `gorm:"index;size:255;not null"`
	Public     bool           `gorm:"index"`
	AppVersion string         `gorm:"size:255;not null"`
	Preset     datatypes.JSON `gorm:"type:jsonb"`
}
