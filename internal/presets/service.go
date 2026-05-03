package presets

import (
	"context"
	"errors"
	"sort"
	"strings"

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
		ID: group.ID,
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

	return &GroupResult{
		ID: groupID,
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

func (s *Service) ListGroups(ctx context.Context, userID uuid.UUID) ([]GroupListItem, error) {
	if userID == uuid.Nil {
		return nil, ErrValidation
	}

	groups, err := gorm.G[models.PresetGroup](s.db).
		Where(g.PresetGroup.UserID.Eq(userID)).
		Order(g.PresetGroup.Name.Asc()).
		Find(ctx)
	if err != nil {
		return nil, ErrInternal
	}

	items := make([]GroupListItem, 0, len(groups))
	for _, group := range groups {
		items = append(items, GroupListItem{
			ID:        group.ID,
			CreatedAt: group.CreatedAt,
			UpdatedAt: group.UpdatedAt,
			UserID:    group.UserID,
			Name:      group.Name,
			Public:    group.Public,
		})
	}

	return items, nil
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
		ID: preset.ID,
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

func (s *Service) ListGroupsWithPresets(ctx context.Context, userID uuid.UUID) ([]GroupWithPresetsItem, error) {
	if userID == uuid.Nil {
		return nil, ErrValidation
	}

	groups, err := gorm.G[models.PresetGroup](s.db).
		Where(g.PresetGroup.UserID.Eq(userID)).
		Order(g.PresetGroup.Name.Asc()).
		Find(ctx)
	if err != nil {
		return nil, ErrInternal
	}

	var groupAlias = g.Preset.Group.Name()

	groupUserID := g.PresetGroup.UserID.WithTable(groupAlias)
	groupName := g.PresetGroup.Name.WithTable(groupAlias)

	presets, err := gorm.G[models.Preset](s.db).
		Joins(clause.InnerJoin.Association(groupAlias), func(db gorm.JoinBuilder, _ clause.Table, _ clause.Table) error {
			db.Where(groupUserID.Eq(userID))
			return nil
		}).
		Order(groupName.Asc()).
		Order(g.Preset.Name.Asc()).
		Find(ctx)
	if err != nil {
		return nil, ErrInternal
	}

	byGroup := make(map[uint][]PresetInGroupTreeItem, len(presets))
	for _, preset := range presets {
		byGroup[preset.GroupId] = append(byGroup[preset.GroupId], PresetInGroupTreeItem{
			ID:         preset.ID,
			CreatedAt:  preset.CreatedAt,
			UpdatedAt:  preset.UpdatedAt,
			GroupID:    preset.GroupId,
			Type:       preset.Type,
			Name:       preset.Name,
			Public:     preset.Public,
			AppVersion: preset.AppVersion,
			Preset:     preset.Preset,
		})
	}

	out := make([]GroupWithPresetsItem, 0, len(groups))
	for _, grp := range groups {
		ps := byGroup[grp.ID]
		if ps == nil {
			ps = []PresetInGroupTreeItem{}
		}
		out = append(out, GroupWithPresetsItem{
			ID:        grp.ID,
			CreatedAt: grp.CreatedAt,
			UpdatedAt: grp.UpdatedAt,
			UserID:    grp.UserID,
			Name:      grp.Name,
			Public:    grp.Public,
			Presets:   ps,
		})
	}

	return out, nil
}

// ListPresetsSharedWithUser returns presets shared with the user via PresetShare and PresetGroupShare,
// nested as owning user → preset group → preset. Owners and groups are ordered by name; presets by name
// within each group. Overlaps between per-preset shares and whole-group shares are deduplicated by preset id.
func (s *Service) ListPresetsSharedWithUser(ctx context.Context, recipientUserID uuid.UUID) ([]SharedPresetsTreeItem, error) {
	if recipientUserID == uuid.Nil {
		return nil, ErrValidation
	}

	presetSharePresetGroupUser := g.PresetShare.Preset.Name() + "." + g.Preset.Group.Name() + "." + g.PresetGroup.User.Name()
	groupShareGroupUser := g.PresetGroupShare.Group.Name() + "." + g.PresetGroup.User.Name()
	omitPassword := func(pb gorm.PreloadBuilder) error {
		pb.Omit(g.User.PasswordHash.Column().Name)
		return nil
	}

	shares, err := gorm.G[models.PresetShare](s.db).
		Where(g.PresetShare.UserID.Eq(recipientUserID)).
		Preload(presetSharePresetGroupUser, omitPassword).
		Find(ctx)
	if err != nil {
		return nil, ErrInternal
	}

	groupShares, err := gorm.G[models.PresetGroupShare](s.db).
		Where(g.PresetGroupShare.UserID.Eq(recipientUserID)).
		Preload(groupShareGroupUser, omitPassword).
		Find(ctx)
	if err != nil {
		return nil, ErrInternal
	}

	type groupBranch struct {
		group      models.PresetGroup
		presets    []PresetInGroupTreeItem
		seenPreset map[uint]struct{}
	}
	type ownerAcc struct {
		owner  SharedPresetOwnerItem
		groups map[uint]*groupBranch
	}

	byOwner := make(map[uuid.UUID]*ownerAcc)

	addPresetDedup := func(gb *groupBranch, item PresetInGroupTreeItem) {
		if gb.seenPreset == nil {
			gb.seenPreset = make(map[uint]struct{})
		}
		if _, dup := gb.seenPreset[item.ID]; dup {
			return
		}
		gb.seenPreset[item.ID] = struct{}{}
		gb.presets = append(gb.presets, item)
	}

	ensureBranch := func(u *models.User, pg *models.PresetGroup) *groupBranch {
		oa, ok := byOwner[u.ID]
		if !ok {
			oa = &ownerAcc{
				owner: SharedPresetOwnerItem{
					ID:        u.ID,
					Username:  u.Username,
					FirstName: u.FirstName,
					LastName:  u.LastName,
				},
				groups: make(map[uint]*groupBranch),
			}
			byOwner[u.ID] = oa
		}
		gb, ok := oa.groups[pg.ID]
		if !ok {
			gb = &groupBranch{group: *pg}
			oa.groups[pg.ID] = gb
		}
		return gb
	}

	presetToItem := func(p *models.Preset) PresetInGroupTreeItem {
		return PresetInGroupTreeItem{
			ID:         p.ID,
			CreatedAt:  p.CreatedAt,
			UpdatedAt:  p.UpdatedAt,
			GroupID:    p.GroupId,
			Type:       p.Type,
			Name:       p.Name,
			Public:     p.Public,
			AppVersion: p.AppVersion,
			Preset:     p.Preset,
		}
	}

	for _, sh := range shares {
		if sh.Preset == nil || sh.Preset.Group == nil || sh.Preset.Group.User == nil {
			continue
		}
		p := sh.Preset
		pg := p.Group
		u := pg.User

		gb := ensureBranch(u, pg)
		addPresetDedup(gb, presetToItem(p))
	}

	distinctGroupIDs := make([]uint, 0, len(groupShares))
	seenGroup := make(map[uint]struct{})
	for _, gsh := range groupShares {
		if gsh.Group == nil {
			continue
		}
		if _, ok := seenGroup[gsh.Group.ID]; ok {
			continue
		}
		seenGroup[gsh.Group.ID] = struct{}{}
		distinctGroupIDs = append(distinctGroupIDs, gsh.Group.ID)
	}

	presetsByGroup := make(map[uint][]models.Preset)
	if len(distinctGroupIDs) > 0 {
		allGroupPresets, err := gorm.G[models.Preset](s.db).
			Where(g.Preset.GroupId.In(distinctGroupIDs...)).
			Order(g.Preset.GroupId.Asc()).
			Order(g.Preset.Name.Asc()).
			Find(ctx)
		if err != nil {
			return nil, ErrInternal
		}
		for i := range allGroupPresets {
			p := allGroupPresets[i]
			presetsByGroup[p.GroupId] = append(presetsByGroup[p.GroupId], p)
		}
	}

	for _, gsh := range groupShares {
		if gsh.Group == nil || gsh.Group.User == nil {
			continue
		}
		pg := gsh.Group
		u := pg.User
		gb := ensureBranch(u, pg)
		for _, p := range presetsByGroup[pg.ID] {
			addPresetDedup(gb, presetToItem(&p))
		}
	}

	ownerIDs := make([]uuid.UUID, 0, len(byOwner))
	for id := range byOwner {
		ownerIDs = append(ownerIDs, id)
	}
	sort.Slice(ownerIDs, func(i, j int) bool {
		a := strings.ToLower(byOwner[ownerIDs[i]].owner.Username)
		b := strings.ToLower(byOwner[ownerIDs[j]].owner.Username)
		if a != b {
			return a < b
		}
		return byOwner[ownerIDs[i]].owner.ID.String() < byOwner[ownerIDs[j]].owner.ID.String()
	})

	out := make([]SharedPresetsTreeItem, 0, len(ownerIDs))
	for _, oid := range ownerIDs {
		oa := byOwner[oid]
		groupIDs := make([]uint, 0, len(oa.groups))
		for gid := range oa.groups {
			groupIDs = append(groupIDs, gid)
		}
		sort.Slice(groupIDs, func(i, j int) bool {
			return oa.groups[groupIDs[i]].group.Name < oa.groups[groupIDs[j]].group.Name
		})

		branches := make([]SharedPresetGroupBranchItem, 0, len(groupIDs))
		for _, gid := range groupIDs {
			gb := oa.groups[gid]
			ps := gb.presets
			if ps == nil {
				ps = []PresetInGroupTreeItem{}
			}
			sort.Slice(ps, func(i, j int) bool {
				return ps[i].Name < ps[j].Name
			})
			branches = append(branches, SharedPresetGroupBranchItem{
				ID:        gb.group.ID,
				CreatedAt: gb.group.CreatedAt,
				UpdatedAt: gb.group.UpdatedAt,
				Name:      gb.group.Name,
				Public:    gb.group.Public,
				Presets:   ps,
			})
		}

		out = append(out, SharedPresetsTreeItem{
			Owner:  oa.owner,
			Groups: branches,
		})
	}

	return out, nil
}

func (s *Service) ListPresetsInGroup(ctx context.Context, userID uuid.UUID, groupID uint) ([]PresetItem, error) {
	if err := validateGroupIdentity(userID, groupID); err != nil {
		return nil, err
	}

	group, err := gorm.G[models.PresetGroup](s.db).
		Select(g.PresetGroup.ID.Column().Name, g.PresetGroup.Name.Column().Name).
		Where(g.PresetGroup.ID.Eq(groupID)).
		Where(g.PresetGroup.UserID.Eq(userID)).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, ErrInternal
	}

	presets, err := gorm.G[models.Preset](s.db).
		Where(g.Preset.GroupId.Eq(groupID)).
		Order(g.Preset.Name.Asc()).
		Find(ctx)
	if err != nil {
		return nil, ErrInternal
	}

	items := make([]PresetItem, 0, len(presets))
	for _, preset := range presets {
		items = append(items, PresetItem{
			ID:         preset.ID,
			CreatedAt:  preset.CreatedAt,
			UpdatedAt:  preset.UpdatedAt,
			GroupID:    preset.GroupId,
			GroupName:  group.Name,
			Type:       preset.Type,
			Name:       preset.Name,
			Public:     preset.Public,
			AppVersion: preset.AppVersion,
			Preset:     preset.Preset,
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

	return &PresetResult{
		ID: input.PresetID,
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

func (s *Service) SharePreset(ctx context.Context, userID uuid.UUID, input SharePresetInput) (*SharePresetResult, error) {
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

	if _, err := gorm.G[models.User](s.db).
		Select(g.User.ID.Column().Name).
		Where(g.User.ID.Eq(input.ShareWithUserID)).
		First(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, ErrInternal
	}

	existingShare, err := gorm.G[models.PresetShare](s.db.Unscoped()).
		Select(
			g.PresetShare.ID.Column().Name,
			g.PresetShare.DeletedAt.Column().Name,
		).
		Where(g.PresetShare.PresetID.Eq(input.PresetID)).
		Where(g.PresetShare.UserID.Eq(input.ShareWithUserID)).
		First(ctx)
	if err == nil {
		if !existingShare.DeletedAt.Valid {
			return &SharePresetResult{ID: existingShare.ID}, nil
		}
		if _, err := gorm.G[models.PresetShare](s.db.Unscoped()).
			Where(g.PresetShare.ID.Eq(existingShare.ID)).
			Set(g.PresetShare.DeletedAt.Set(gorm.DeletedAt{})).
			Update(ctx); err != nil {
			return nil, ErrInternal
		}
		return &SharePresetResult{ID: existingShare.ID}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInternal
	}

	share := models.PresetShare{
		PresetID: input.PresetID,
		UserID:   input.ShareWithUserID,
	}
	if err := gorm.G[models.PresetShare](s.db).Create(ctx, &share); err != nil {
		return nil, ErrInternal
	}

	return &SharePresetResult{ID: share.ID}, nil
}

func (s *Service) ShareGroup(ctx context.Context, userID uuid.UUID, input ShareGroupInput) (*ShareGroupResult, error) {
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

	if _, err := gorm.G[models.User](s.db).
		Select(g.User.ID.Column().Name).
		Where(g.User.ID.Eq(input.ShareWithUserID)).
		First(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, ErrInternal
	}

	existingShare, err := gorm.G[models.PresetGroupShare](s.db.Unscoped()).
		Select(
			g.PresetGroupShare.ID.Column().Name,
			g.PresetGroupShare.DeletedAt.Column().Name,
		).
		Where(g.PresetGroupShare.GroupID.Eq(input.GroupID)).
		Where(g.PresetGroupShare.UserID.Eq(input.ShareWithUserID)).
		First(ctx)
	if err == nil {
		if !existingShare.DeletedAt.Valid {
			return &ShareGroupResult{ID: existingShare.ID}, nil
		}
		if _, err := gorm.G[models.PresetGroupShare](s.db.Unscoped()).
			Where(g.PresetGroupShare.ID.Eq(existingShare.ID)).
			Set(g.PresetGroupShare.DeletedAt.Set(gorm.DeletedAt{})).
			Update(ctx); err != nil {
			return nil, ErrInternal
		}
		return &ShareGroupResult{ID: existingShare.ID}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInternal
	}

	share := models.PresetGroupShare{
		GroupID: input.GroupID,
		UserID:  input.ShareWithUserID,
	}
	if err := gorm.G[models.PresetGroupShare](s.db).Create(ctx, &share); err != nil {
		return nil, ErrInternal
	}

	return &ShareGroupResult{ID: share.ID}, nil
}
