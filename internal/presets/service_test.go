package presets

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/alekseikl/additizer-api/internal/models"
	"github.com/alekseikl/additizer-api/internal/testdb"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func newTestPresetService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := testdb.Open(t)
	return NewService(db), db
}

func mustCreateUser(t *testing.T, db *gorm.DB, username, email string) uuid.UUID {
	t.Helper()
	u := models.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		FirstName:    "First",
		LastName:     "Last",
		PasswordHash: "hash",
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u.ID
}

func mustCreateGroup(t *testing.T, db *gorm.DB, userID uuid.UUID, name string) uint {
	t.Helper()
	g := models.PresetGroup{
		UserID: userID,
		Name:   name,
		Public: false,
	}
	if err := db.Create(&g).Error; err != nil {
		t.Fatalf("create preset group: %v", err)
	}
	return g.ID
}

func validPresetJSON() datatypes.JSON {
	return datatypes.JSON([]byte(`{"k":"v"}`))
}

func assertEqualJSON(t *testing.T, got, want []byte) {
	t.Helper()
	var a, b any
	if err := json.Unmarshal(got, &a); err != nil {
		t.Fatalf("parse got JSON: %v", err)
	}
	if err := json.Unmarshal(want, &b); err != nil {
		t.Fatalf("parse want JSON: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("JSON mismatch: got %s want %s", got, want)
	}
}

func TestCreateGroupCreatesRow(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")

	res, err := svc.CreateGroup(ctx, owner, CreateGroupInput{Name: "  My Group  ", Public: true})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if res == nil || res.ID == 0 {
		t.Fatalf("expected group id, got %#v", res)
	}

	var stored models.PresetGroup
	if err := db.First(&stored, res.ID).Error; err != nil {
		t.Fatalf("load group: %v", err)
	}
	if stored.UserID != owner || stored.Name != "My Group" || !stored.Public {
		t.Fatalf("unexpected stored group: %#v", stored)
	}
}

func TestCreateGroupValidation(t *testing.T) {
	svc, _ := newTestPresetService(t)
	ctx := context.Background()
	owner := uuid.New()

	_, err := svc.CreateGroup(ctx, uuid.Nil, CreateGroupInput{Name: "G"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}

	_, err = svc.CreateGroup(ctx, owner, CreateGroupInput{Name: ""})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestUpdateGroupChangesFields(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "Original")

	res, err := svc.UpdateGroup(ctx, owner, gid, UpdateGroupInput{Name: "  Renamed  ", Public: true})
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if res.ID != gid {
		t.Fatalf("expected id %d, got %d", gid, res.ID)
	}

	var stored models.PresetGroup
	if err := db.First(&stored, gid).Error; err != nil {
		t.Fatalf("load group: %v", err)
	}
	if stored.Name != "Renamed" || !stored.Public {
		t.Fatalf("unexpected stored group: %#v", stored)
	}
}

func TestUpdateGroupNotFoundForOtherUser(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	alice := mustCreateUser(t, db, "alice", "a@example.com")
	bob := mustCreateUser(t, db, "bob", "b@example.com")
	gid := mustCreateGroup(t, db, alice, "Alice Group")

	_, err := svc.UpdateGroup(ctx, bob, gid, UpdateGroupInput{Name: "Hijack", Public: false})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteGroupRemovesRow(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "ToDelete")

	if err := svc.DeleteGroup(ctx, owner, gid); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	var count int64
	if err := db.Model(&models.PresetGroup{}).Where("id = ?", gid).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected group deleted, count=%d", count)
	}
}

func TestDeleteGroupNotFound(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")

	err := svc.DeleteGroup(ctx, owner, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListGroupsOrdersByName(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	mustCreateGroup(t, db, owner, "Beta")
	mustCreateGroup(t, db, owner, "Alpha")

	items, err := svc.ListGroups(ctx, owner)
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(items))
	}
	if items[0].Name != "Alpha" || items[1].Name != "Beta" {
		t.Fatalf("unexpected order: %#v", items)
	}
}

func TestListGroupsNilUser(t *testing.T) {
	svc, _ := newTestPresetService(t)
	ctx := context.Background()

	_, err := svc.ListGroups(ctx, uuid.Nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreatePresetCreatesRow(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "G")

	res, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Oscillator,
		Name:       "  P ",
		Public:     true,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}
	if res == nil || res.ID == 0 {
		t.Fatalf("expected preset id, got %#v", res)
	}

	var stored models.Preset
	if err := db.First(&stored, res.ID).Error; err != nil {
		t.Fatalf("load preset: %v", err)
	}
	if stored.GroupId != gid || stored.Name != "P" || stored.Type != models.Oscillator || !stored.Public {
		t.Fatalf("unexpected stored preset: %#v", stored)
	}
}

func TestCreatePresetGroupMissing(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")

	_, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    99999,
		Type:       models.Oscillator,
		Name:       "P",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreatePresetWrongOwner(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	alice := mustCreateUser(t, db, "alice", "a@example.com")
	bob := mustCreateUser(t, db, "bob", "b@example.com")
	gid := mustCreateGroup(t, db, alice, "Alice only")

	_, err := svc.CreatePreset(ctx, bob, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Oscillator,
		Name:       "P",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListPresetsIncludesGroupName(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "MyGrp")
	_, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.HarmonicsEditor,
		Name:       "P1",
		Public:     false,
		AppVersion: "2.1.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	items, err := svc.ListPresets(ctx, owner)
	if err != nil {
		t.Fatalf("ListPresets: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 preset, got %d", len(items))
	}
	if items[0].GroupName != "MyGrp" || items[0].Name != "P1" || items[0].AppVersion != "2.1.0" {
		t.Fatalf("unexpected list item: %#v", items[0])
	}
}

func TestListPresetsNilUser(t *testing.T) {
	svc, _ := newTestPresetService(t)
	ctx := context.Background()

	_, err := svc.ListPresets(ctx, uuid.Nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestListGroupsWithPresetsEmptyPresetSlice(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "G")
	_, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Mixer,
		Name:       "Only",
		Public:     true,
		AppVersion: "1.2.3",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}
	emptyGid := mustCreateGroup(t, db, owner, "Empty")

	out, err := svc.ListGroupsWithPresets(ctx, owner)
	if err != nil {
		t.Fatalf("ListGroupsWithPresets: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(out))
	}
	var empty *GroupWithPresetsItem
	for i := range out {
		if out[i].ID == emptyGid {
			empty = &out[i]
			break
		}
	}
	if empty == nil {
		t.Fatal("empty group not found")
	}
	if empty.Presets == nil || len(empty.Presets) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", empty.Presets)
	}
}

func TestListPresetsInGroup(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "Grp")
	_, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Envelope,
		Name:       "E",
		Public:     false,
		AppVersion: "0.1.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	items, err := svc.ListPresetsInGroup(ctx, owner, gid)
	if err != nil {
		t.Fatalf("ListPresetsInGroup: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 preset, got %d", len(items))
	}
	if items[0].GroupName != "Grp" || items[0].Name != "E" {
		t.Fatalf("unexpected item: %#v", items[0])
	}
	assertEqualJSON(t, items[0].Preset, validPresetJSON())
}

func TestListPresetsInGroupNotFound(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")

	_, err := svc.ListPresetsInGroup(ctx, owner, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdatePresetNameOnly(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "G")
	createRes, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Oscillator,
		Name:       "Old",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	res, err := svc.UpdatePreset(ctx, owner, UpdatePresetInput{
		PresetID: createRes.ID,
		Type:     models.Oscillator,
		Name:     "  New Name ",
		Public:   true,
	})
	if err != nil {
		t.Fatalf("UpdatePreset: %v", err)
	}
	if res.ID != createRes.ID {
		t.Fatalf("unexpected id %d", res.ID)
	}

	var stored models.Preset
	if err := db.First(&stored, createRes.ID).Error; err != nil {
		t.Fatalf("load preset: %v", err)
	}
	if stored.Name != "New Name" || !stored.Public {
		t.Fatalf("unexpected stored: %#v", stored)
	}
}

func TestUpdatePresetWithAppVersionAndJSON(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "G")
	createRes, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Waveshaper,
		Name:       "W",
		Public:     true,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	newVer := "3.0.0"
	newBody := datatypes.JSON([]byte(`{"x":true}`))
	if _, err := svc.UpdatePreset(ctx, owner, UpdatePresetInput{
		PresetID:   createRes.ID,
		Type:       models.Waveshaper,
		Name:       "W2",
		Public:     false,
		AppVersion: &newVer,
		Preset:     &newBody,
	}); err != nil {
		t.Fatalf("UpdatePreset: %v", err)
	}

	var stored models.Preset
	if err := db.First(&stored, createRes.ID).Error; err != nil {
		t.Fatalf("load preset: %v", err)
	}
	if stored.AppVersion != "3.0.0" {
		t.Fatalf("unexpected stored app version %#v", stored.AppVersion)
	}
	assertEqualJSON(t, stored.Preset, []byte(`{"x":true}`))
}

func TestDeletePreset(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "G")
	createRes, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Amplifier,
		Name:       "A",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	if err := svc.DeletePreset(ctx, owner, createRes.ID); err != nil {
		t.Fatalf("DeletePreset: %v", err)
	}
	var count int64
	if err := db.Model(&models.Preset{}).Where("id = ?", createRes.ID).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected preset deleted")
	}
}

func TestSharePresetCreatesRow(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	recipient := mustCreateUser(t, db, "bob", "b@example.com")
	gid := mustCreateGroup(t, db, owner, "G")
	createRes, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Mixer,
		Name:       "M",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	res, err := svc.SharePreset(ctx, owner, SharePresetInput{
		PresetID:        createRes.ID,
		ShareWithUserID: recipient,
	})
	if err != nil {
		t.Fatalf("SharePreset: %v", err)
	}
	if res == nil || res.ID == 0 {
		t.Fatalf("unexpected result %#v", res)
	}

	var share models.PresetShare
	if err := db.First(&share, res.ID).Error; err != nil {
		t.Fatalf("load share: %v", err)
	}
	if share.PresetID != createRes.ID || share.UserID != recipient {
		t.Fatalf("unexpected share: %#v", share)
	}
}

func TestSharePresetRejectsSelf(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "G")
	createRes, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Mixer,
		Name:       "M",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	_, err = svc.SharePreset(ctx, owner, SharePresetInput{
		PresetID:        createRes.ID,
		ShareWithUserID: owner,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestSharePresetRecipientMissing(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	gid := mustCreateGroup(t, db, owner, "G")
	createRes, err := svc.CreatePreset(ctx, owner, CreatePresetInput{
		GroupID:    gid,
		Type:       models.Mixer,
		Name:       "M",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	_, err = svc.SharePreset(ctx, owner, SharePresetInput{
		PresetID:        createRes.ID,
		ShareWithUserID: uuid.New(),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestShareGroupCreatesRow(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	owner := mustCreateUser(t, db, "alice", "a@example.com")
	recipient := mustCreateUser(t, db, "bob", "b@example.com")
	gid := mustCreateGroup(t, db, owner, "G")

	res, err := svc.ShareGroup(ctx, owner, ShareGroupInput{
		GroupID:         gid,
		ShareWithUserID: recipient,
	})
	if err != nil {
		t.Fatalf("ShareGroup: %v", err)
	}
	if res == nil || res.ID == 0 {
		t.Fatalf("unexpected result %#v", res)
	}

	var share models.PresetGroupShare
	if err := db.First(&share, res.ID).Error; err != nil {
		t.Fatalf("load share: %v", err)
	}
	if share.GroupID != gid || share.UserID != recipient {
		t.Fatalf("unexpected share: %#v", share)
	}
}

func TestListPresetsSharedWithUserDeduplicatesAndOrders(t *testing.T) {
	svc, db := newTestPresetService(t)
	ctx := context.Background()
	alice := mustCreateUser(t, db, "zebra", "zebra@example.com")
	bob := mustCreateUser(t, db, "apple", "apple@example.com")
	recipient := mustCreateUser(t, db, "carl", "carl@example.com")

	gA := mustCreateGroup(t, db, alice, "AliceGrp")
	p1, err := svc.CreatePreset(ctx, alice, CreatePresetInput{
		GroupID:    gA,
		Type:       models.Oscillator,
		Name:       "Zeta",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}
	_, err = svc.CreatePreset(ctx, alice, CreatePresetInput{
		GroupID:    gA,
		Type:       models.Oscillator,
		Name:       "Alpha",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	if _, err := svc.SharePreset(ctx, alice, SharePresetInput{PresetID: p1.ID, ShareWithUserID: recipient}); err != nil {
		t.Fatalf("SharePreset: %v", err)
	}
	if _, err := svc.ShareGroup(ctx, alice, ShareGroupInput{GroupID: gA, ShareWithUserID: recipient}); err != nil {
		t.Fatalf("ShareGroup: %v", err)
	}

	gB := mustCreateGroup(t, db, bob, "BobGrp")
	_, err = svc.CreatePreset(ctx, bob, CreatePresetInput{
		GroupID:    gB,
		Type:       models.Envelope,
		Name:       "BOnly",
		Public:     false,
		AppVersion: "1.0.0",
		Preset:     validPresetJSON(),
	})
	if err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}
	if _, err := svc.ShareGroup(ctx, bob, ShareGroupInput{GroupID: gB, ShareWithUserID: recipient}); err != nil {
		t.Fatalf("ShareGroup bob: %v", err)
	}

	out, err := svc.ListPresetsSharedWithUser(ctx, recipient)
	if err != nil {
		t.Fatalf("ListPresetsSharedWithUser: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 owners, got %d", len(out))
	}
	// apple < zebra by username (case-insensitive)
	if out[0].Owner.Username != "apple" || out[1].Owner.Username != "zebra" {
		t.Fatalf("unexpected owner order: %#v, %#v", out[0].Owner, out[1].Owner)
	}

	aliceBlock := out[1]
	if len(aliceBlock.Groups) != 1 || aliceBlock.Groups[0].Name != "AliceGrp" {
		t.Fatalf("unexpected alice groups: %#v", aliceBlock.Groups)
	}
	presets := aliceBlock.Groups[0].Presets
	if len(presets) != 2 {
		t.Fatalf("expected 2 presets after dedup, got %d", len(presets))
	}
	if presets[0].Name != "Alpha" || presets[1].Name != "Zeta" {
		t.Fatalf("unexpected preset order: %#v", presets)
	}
}

func TestListPresetsSharedWithUserNilRecipient(t *testing.T) {
	svc, _ := newTestPresetService(t)
	ctx := context.Background()

	_, err := svc.ListPresetsSharedWithUser(ctx, uuid.Nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
