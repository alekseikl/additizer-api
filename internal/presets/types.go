package presets

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/alekseikl/additizer-api/internal/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var (
	ErrValidation = errors.New("validation failed")
	ErrInternal   = errors.New("internal")
	ErrNotFound   = errors.New("not found")
)

type CreateGroupInput struct {
	Name   string
	Public bool
}

func (i *CreateGroupInput) normalize() {
	i.Name = strings.TrimSpace(i.Name)
}

func (i *CreateGroupInput) validate(userID uuid.UUID) error {
	if userID == uuid.Nil {
		return fmt.Errorf("%w: user id is required", ErrValidation)
	}
	if err := validateGroupName(i.Name); err != nil {
		return err
	}
	return nil
}

type UpdateGroupInput struct {
	Name   string
	Public bool
}

func (i *UpdateGroupInput) normalize() {
	i.Name = strings.TrimSpace(i.Name)
}

func (i *UpdateGroupInput) validate(userID uuid.UUID, groupID uint) error {
	if err := validateGroupIdentity(userID, groupID); err != nil {
		return err
	}
	if err := validateGroupName(i.Name); err != nil {
		return err
	}
	return nil
}

func validateGroupIdentity(userID uuid.UUID, groupID uint) error {
	if userID == uuid.Nil {
		return fmt.Errorf("%w: user id is required", ErrValidation)
	}
	if groupID == 0 {
		return fmt.Errorf("%w: group id is required", ErrValidation)
	}
	return nil
}

func validateGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if len(name) > 255 {
		return fmt.Errorf("%w: name must be at most 255 characters", ErrValidation)
	}
	return nil
}

type CreatePresetInput struct {
	GroupID    uint
	Type       models.ModuleType
	Name       string
	Public     bool
	AppVersion string
	Preset     datatypes.JSON
}

type UpdatePresetInput struct {
	PresetID   uint
	Type       models.ModuleType
	Name       string
	Public     bool
	AppVersion *string
	Preset     *datatypes.JSON
}

func (i *CreatePresetInput) normalize() {
	i.Name = strings.TrimSpace(i.Name)
	i.AppVersion = strings.TrimSpace(i.AppVersion)
}

func (i *UpdatePresetInput) normalize() {
	i.Name = strings.TrimSpace(i.Name)
	if i.AppVersion != nil {
		appVersion := strings.TrimSpace(*i.AppVersion)
		i.AppVersion = &appVersion
	}
}

func (i *CreatePresetInput) validate(userID uuid.UUID) error {
	if err := validateGroupIdentity(userID, i.GroupID); err != nil {
		return err
	}
	if err := validatePresetType(i.Type); err != nil {
		return err
	}
	if err := validatePresetName(i.Name); err != nil {
		return err
	}
	if err := validateAppVersion(i.AppVersion); err != nil {
		return err
	}
	if err := validatePresetData(i.Preset); err != nil {
		return err
	}
	return nil
}

func (i *UpdatePresetInput) validate(userID uuid.UUID) error {
	if err := validatePresetIdentity(userID, i.PresetID); err != nil {
		return err
	}
	if err := validatePresetType(i.Type); err != nil {
		return err
	}
	if err := validatePresetName(i.Name); err != nil {
		return err
	}
	if (i.AppVersion == nil) != (i.Preset == nil) {
		return fmt.Errorf("%w: app version and preset must be provided together", ErrValidation)
	}
	if i.AppVersion != nil {
		if err := validateAppVersion(*i.AppVersion); err != nil {
			return err
		}
		if err := validatePresetData(*i.Preset); err != nil {
			return err
		}
	}
	return nil
}

func validatePresetIdentity(userID uuid.UUID, presetID uint) error {
	if userID == uuid.Nil {
		return fmt.Errorf("%w: user id is required", ErrValidation)
	}
	if presetID == 0 {
		return fmt.Errorf("%w: preset id is required", ErrValidation)
	}
	return nil
}

func validateAppVersion(appVersion string) error {
	if appVersion == "" {
		return fmt.Errorf("%w: app version is required", ErrValidation)
	}
	if len(appVersion) > 255 {
		return fmt.Errorf("%w: app version must be at most 255 characters", ErrValidation)
	}
	if _, err := semver.StrictNewVersion(appVersion); err != nil {
		return fmt.Errorf("%w: app version must be a valid semver string", ErrValidation)
	}
	return nil
}

func validatePresetData(preset datatypes.JSON) error {
	if len(preset) == 0 {
		return fmt.Errorf("%w: preset is required", ErrValidation)
	}
	if !json.Valid(preset) {
		return fmt.Errorf("%w: preset must be valid JSON", ErrValidation)
	}
	return nil
}

func validatePresetType(moduleType models.ModuleType) error {
	switch moduleType {
	case models.HarmonicsEditor,
		models.Oscillator,
		models.SpectralFilter,
		models.Envelope,
		models.SpectralMixer,
		models.SpectralBlend,
		models.Mixer,
		models.Amplifier,
		models.Waveshaper:
		return nil
	default:
		return fmt.Errorf("%w: invalid preset type", ErrValidation)
	}
}

func validatePresetName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if len(name) > 255 {
		return fmt.Errorf("%w: name must be at most 255 characters", ErrValidation)
	}
	return nil
}

type GroupResult struct {
	ID uint
}

type GroupListItem struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uuid.UUID
	Name      string
	Public    bool
}

type SharePresetInput struct {
	PresetID        uint
	ShareWithUserID uuid.UUID
}

func (i *SharePresetInput) normalize() {}

func (i *SharePresetInput) validate(ownerUserID uuid.UUID) error {
	if err := validatePresetIdentity(ownerUserID, i.PresetID); err != nil {
		return err
	}
	if i.ShareWithUserID == uuid.Nil {
		return fmt.Errorf("%w: share recipient user id is required", ErrValidation)
	}
	if i.ShareWithUserID == ownerUserID {
		return fmt.Errorf("%w: cannot share a preset with yourself", ErrValidation)
	}
	return nil
}

type PresetResult struct {
	ID uint
}

type SharePresetResult struct {
	ID uint
}

type ShareGroupInput struct {
	GroupID         uint
	ShareWithUserID uuid.UUID
}

func (i *ShareGroupInput) normalize() {}

func (i *ShareGroupInput) validate(ownerUserID uuid.UUID) error {
	if err := validateGroupIdentity(ownerUserID, i.GroupID); err != nil {
		return err
	}
	if i.ShareWithUserID == uuid.Nil {
		return fmt.Errorf("%w: share recipient user id is required", ErrValidation)
	}
	if i.ShareWithUserID == ownerUserID {
		return fmt.Errorf("%w: cannot share a preset group with yourself", ErrValidation)
	}
	return nil
}

type ShareGroupResult struct {
	ID uint
}

type PresetListItem struct {
	ID         uint
	CreatedAt  time.Time
	UpdatedAt  time.Time
	GroupID    uint
	GroupName  string
	Type       models.ModuleType
	Name       string
	Public     bool
	AppVersion string
}

type PresetItem struct {
	ID         uint
	CreatedAt  time.Time
	UpdatedAt  time.Time
	GroupID    uint
	GroupName  string
	Type       models.ModuleType
	Name       string
	Public     bool
	AppVersion string
	Preset     datatypes.JSON
}

type GroupWithPresetsItem struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uuid.UUID
	Name      string
	Public    bool
	Presets   []PresetInGroupTreeItem
}

type PresetInGroupTreeItem struct {
	ID         uint
	CreatedAt  time.Time
	UpdatedAt  time.Time
	GroupID    uint
	Type       models.ModuleType
	Name       string
	Public     bool
	AppVersion string
	Preset     datatypes.JSON
}

// SharedPresetsTreeItem is owner user → preset groups → presets shared with the recipient.
type SharedPresetsTreeItem struct {
	Owner  SharedPresetOwnerItem
	Groups []SharedPresetGroupBranchItem
}

// SharedPresetOwnerItem is the preset owner's public profile (no credentials).
type SharedPresetOwnerItem struct {
	ID        uuid.UUID
	Username  string
	FirstName string
	LastName  string
}

// SharedPresetGroupBranchItem is a preset group belonging to Owner with only presets shared with the recipient.
type SharedPresetGroupBranchItem struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
	Public    bool
	Presets   []PresetInGroupTreeItem
}
