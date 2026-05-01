package presets

import (
	"context"
	"errors"

	g "github.com/alekseikl/additizer-api/internal/generated"
	"github.com/alekseikl/additizer-api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateGroup(ctx context.Context, userID uuid.UUID, input CreateGroupInput) (*GroupResult, error) {
	input.normalize()

	if err := input.validate(userID); err != nil {
		return nil, err
	}

	group := models.PresetGroup{
		UserID: userID,
		Name:   input.Name,
		Public: input.Public,
	}

	if err := gorm.G[models.PresetGroup](s.db).Create(ctx, &group); err != nil {
		return nil, ErrInternal
	}

	return &GroupResult{
		ID:     group.ID,
		UserID: group.UserID,
		Name:   group.Name,
		Public: group.Public,
	}, nil
}

func (s *Service) UpdateGroup(ctx context.Context, userID uuid.UUID, groupID uint, input UpdateGroupInput) (*GroupResult, error) {
	input.normalize()

	if err := input.validate(userID, groupID); err != nil {
		return nil, err
	}

	if _, err := gorm.G[models.PresetGroup](s.db).
		Select(g.PresetGroup.ID.Column().Name).
		Where(g.PresetGroup.ID.Eq(groupID)).
		Where(g.PresetGroup.UserID.Eq(userID)).
		First(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, ErrInternal
	}

	if _, err := gorm.G[models.PresetGroup](s.db).
		Where(g.PresetGroup.ID.Eq(groupID)).
		Where(g.PresetGroup.UserID.Eq(userID)).
		Set(
			g.PresetGroup.Name.Set(input.Name),
			g.PresetGroup.Public.Set(input.Public),
		).
		Update(ctx); err != nil {
		return nil, ErrInternal
	}

	group, err := gorm.G[models.PresetGroup](s.db).
		Where(g.PresetGroup.ID.Eq(groupID)).
		Where(g.PresetGroup.UserID.Eq(userID)).
		First(ctx)
	if err != nil {
		return nil, ErrInternal
	}

	return &GroupResult{
		ID:     group.ID,
		UserID: group.UserID,
		Name:   group.Name,
		Public: group.Public,
	}, nil
}

func (s *Service) DeleteGroup(ctx context.Context, userID uuid.UUID, groupID uint) error {
	if err := validateGroupIdentity(userID, groupID); err != nil {
		return err
	}

	if _, err := gorm.G[models.PresetGroup](s.db).
		Select(g.PresetGroup.ID.Column().Name).
		Where(g.PresetGroup.ID.Eq(groupID)).
		Where(g.PresetGroup.UserID.Eq(userID)).
		First(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return ErrInternal
	}

	if _, err := gorm.G[models.PresetGroup](s.db).
		Where(g.PresetGroup.ID.Eq(groupID)).
		Where(g.PresetGroup.UserID.Eq(userID)).
		Delete(ctx); err != nil {
		return ErrInternal
	}

	return nil
}

func (s *Service) CreatePreset(ctx context.Context, userID uuid.UUID, input CreatePresetInput) (*PresetResult, error) {
	input.normalize()

	if err := input.validate(userID); err != nil {
		return nil, err
	}

	if _, err := gorm.G[models.PresetGroup](s.db).
		Select(g.PresetGroup.ID.Column().Name).
		Where(g.PresetGroup.ID.Eq(input.GroupID)).
		Where(g.PresetGroup.UserID.Eq(userID)).
		First(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, ErrInternal
	}

	preset := models.Preset{
		GroupId:    input.GroupID,
		Type:       input.Type,
		Name:       input.Name,
		Public:     input.Public,
		AppVersion: input.AppVersion,
		Preset:     input.Preset,
	}

	if err := gorm.G[models.Preset](s.db).Create(ctx, &preset); err != nil {
		return nil, ErrInternal
	}

	return &PresetResult{
		ID:         preset.ID,
		GroupID:    preset.GroupId,
		Type:       preset.Type,
		Name:       preset.Name,
		Public:     preset.Public,
		AppVersion: preset.AppVersion,
	}, nil
}

func (s *Service) UpdatePreset(ctx context.Context, userID uuid.UUID, input UpdatePresetInput) (*PresetResult, error) {
	input.normalize()

	if err := input.validate(userID); err != nil {
		return nil, err
	}

	existing, err := gorm.G[models.Preset](s.db).
		Select(g.Preset.ID.Column().Name, g.Preset.GroupId.Column().Name).
		Where(g.Preset.ID.Eq(input.PresetID)).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, ErrInternal
	}

	if _, err := gorm.G[models.PresetGroup](s.db).
		Select(g.PresetGroup.ID.Column().Name).
		Where(g.PresetGroup.ID.Eq(existing.GroupId)).
		Where(g.PresetGroup.UserID.Eq(userID)).
		First(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, ErrInternal
	}

	assigners := []clause.Assigner{
		clause.Assignment{Column: clause.Column{Name: "type"}, Value: input.Type},
		g.Preset.Name.Set(input.Name),
		g.Preset.Public.Set(input.Public),
	}
	if input.AppVersion != nil {
		assigners = append(assigners,
			g.Preset.AppVersion.Set(*input.AppVersion),
			g.Preset.Preset.Set(*input.Preset),
		)
	}

	if _, err := gorm.G[models.Preset](s.db).
		Where(g.Preset.ID.Eq(input.PresetID)).
		Set(assigners...).
		Update(ctx); err != nil {
		return nil, ErrInternal
	}

	preset, err := gorm.G[models.Preset](s.db).
		Where(g.Preset.ID.Eq(input.PresetID)).
		First(ctx)
	if err != nil {
		return nil, ErrInternal
	}

	return &PresetResult{
		ID:         preset.ID,
		GroupID:    preset.GroupId,
		Type:       preset.Type,
		Name:       preset.Name,
		Public:     preset.Public,
		AppVersion: preset.AppVersion,
	}, nil
}

func (s *Service) DeletePreset(ctx context.Context, userID uuid.UUID, presetID uint) error {
	if err := validatePresetIdentity(userID, presetID); err != nil {
		return err
	}

	existing, err := gorm.G[models.Preset](s.db).
		Select(g.Preset.ID.Column().Name, g.Preset.GroupId.Column().Name).
		Where(g.Preset.ID.Eq(presetID)).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return ErrInternal
	}

	if _, err := gorm.G[models.PresetGroup](s.db).
		Select(g.PresetGroup.ID.Column().Name).
		Where(g.PresetGroup.ID.Eq(existing.GroupId)).
		Where(g.PresetGroup.UserID.Eq(userID)).
		First(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return ErrInternal
	}

	if _, err := gorm.G[models.Preset](s.db).
		Where(g.Preset.ID.Eq(presetID)).
		Delete(ctx); err != nil {
		return ErrInternal
	}

	return nil
}
