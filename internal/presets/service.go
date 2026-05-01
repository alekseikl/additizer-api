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

func (s *Service) ListPresets(ctx context.Context, userID uuid.UUID) ([]PresetListItem, error) {
	if userID == uuid.Nil {
		return nil, ErrValidation
	}

	var groupAlias = g.Preset.Group.Name()

	groupUserID := g.PresetGroup.UserID.WithTable(groupAlias)
	groupName := g.PresetGroup.Name.WithTable(groupAlias)

	presets, err := gorm.G[models.Preset](s.db).
		Joins(clause.InnerJoin.Association(groupAlias), func(db gorm.JoinBuilder, _ clause.Table, _ clause.Table) error {
			db.Where(groupUserID.Eq(userID))
			return nil
		}).
		Omit(g.Preset.Preset.Column().Name).
		Order(groupName.Asc()).
		Order(g.Preset.Name.Asc()).
		Find(ctx)
	if err != nil {
		return nil, ErrInternal
	}

	items := make([]PresetListItem, 0, len(presets))
	for _, preset := range presets {
		name := ""
		if preset.Group != nil {
			name = preset.Group.Name
		}
		items = append(items, PresetListItem{
			ID:         preset.ID,
			CreatedAt:  preset.CreatedAt,
			UpdatedAt:  preset.UpdatedAt,
			GroupID:    preset.GroupId,
			GroupName:  name,
			Type:       preset.Type,
			Name:       preset.Name,
			Public:     preset.Public,
			AppVersion: preset.AppVersion,
		})
	}

	return items, nil
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

// func (s *Service) Check(ctx context.Context) {
// 	s.ListPresets(ctx, uuid.MustParse("c5afb29a-c07f-4b43-b673-e1a1b1271819"))
// }
