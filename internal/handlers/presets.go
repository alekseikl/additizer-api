package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/alekseikl/additizer-api/internal/httpx"
	"github.com/alekseikl/additizer-api/internal/middleware"
	"github.com/alekseikl/additizer-api/internal/models"
	"github.com/alekseikl/additizer-api/internal/presets"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type PresetsHandler struct {
	presets *presets.Service
}

func NewPresetsHandler(presets *presets.Service) *PresetsHandler {
	return &PresetsHandler{presets: presets}
}

type createGroupRequest struct {
	Name   string `json:"name"`
	Public bool   `json:"public"`
}

type updateGroupRequest struct {
	Name   string `json:"name"`
	Public bool   `json:"public"`
}

type createPresetRequest struct {
	GroupID    uint              `json:"group_id"`
	Type       models.ModuleType `json:"type"`
	Name       string            `json:"name"`
	Public     bool              `json:"public"`
	AppVersion string            `json:"app_version"`
	Preset     json.RawMessage   `json:"preset"`
}

type updatePresetRequest struct {
	Type       models.ModuleType `json:"type"`
	Name       string            `json:"name"`
	Public     bool              `json:"public"`
	AppVersion *string           `json:"app_version"`
	Preset     *json.RawMessage  `json:"preset"`
}

type groupResultResponse struct {
	ID uint `json:"id"`
}

type presetResultResponse struct {
	ID uint `json:"id"`
}

type shareWithUserRequest struct {
	ShareWithUserID uuid.UUID `json:"share_with_user_id"`
}

type shareRecordResponse struct {
	ID uint `json:"id"`
}

type presetInGroupResponse struct {
	ID         uint              `json:"id"`
	CreatedAt  *time.Time        `json:"created_at,omitempty"`
	UpdatedAt  *time.Time        `json:"updated_at,omitempty"`
	GroupID    uint              `json:"group_id"`
	Type       models.ModuleType `json:"type"`
	Name       string            `json:"name"`
	Public     bool              `json:"public"`
	AppVersion string            `json:"app_version"`
	Preset     json.RawMessage   `json:"preset"`
}

type groupWithPresetsResponse struct {
	ID        uint                    `json:"id"`
	CreatedAt *time.Time              `json:"created_at,omitempty"`
	UpdatedAt *time.Time              `json:"updated_at,omitempty"`
	UserID    string                  `json:"user_id"`
	Name      string                  `json:"name"`
	Public    bool                    `json:"public"`
	Presets   []presetInGroupResponse `json:"presets"`
}

type sharedPresetOwnerResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type sharedPresetGroupResponse struct {
	ID        uint                    `json:"id"`
	CreatedAt *time.Time              `json:"created_at,omitempty"`
	UpdatedAt *time.Time              `json:"updated_at,omitempty"`
	Name      string                  `json:"name"`
	Public    bool                    `json:"public"`
	Presets   []presetInGroupResponse `json:"presets"`
}

type sharedPresetsResponse struct {
	Owner  sharedPresetOwnerResponse   `json:"owner"`
	Groups []sharedPresetGroupResponse `json:"groups"`
}

type groupResponse struct {
	ID        uint       `json:"id"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	UserID    string     `json:"user_id"`
	Name      string     `json:"name"`
	Public    bool       `json:"public"`
}

type presetResponse struct {
	ID         uint              `json:"id"`
	CreatedAt  *time.Time        `json:"created_at,omitempty"`
	UpdatedAt  *time.Time        `json:"updated_at,omitempty"`
	GroupID    uint              `json:"group_id"`
	GroupName  string            `json:"group_name,omitempty"`
	Type       models.ModuleType `json:"type"`
	Name       string            `json:"name"`
	Public     bool              `json:"public"`
	AppVersion string            `json:"app_version"`
	Preset     json.RawMessage   `json:"preset,omitempty"`
}

func (h *PresetsHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	req, err := httpx.DecodeJSON[createGroupRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	result, err := h.presets.CreateGroup(ctx, userID, presets.CreateGroupInput{
		Name:   req.Name,
		Public: req.Public,
	})
	if err != nil {
		writePresetsError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, groupResultResponse{
		ID: result.ID,
	})
}

func (h *PresetsHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	groupID, ok := httpx.UintURLParam(w, r, "groupID")
	if !ok {
		return
	}

	req, err := httpx.DecodeJSON[updateGroupRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	result, err := h.presets.UpdateGroup(ctx, userID, groupID, presets.UpdateGroupInput{
		Name:   req.Name,
		Public: req.Public,
	})
	if err != nil {
		writePresetsError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, groupResultResponse{
		ID: result.ID,
	})
}

func (h *PresetsHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	groupID, ok := httpx.UintURLParam(w, r, "groupID")
	if !ok {
		return
	}

	if err := h.presets.DeleteGroup(ctx, userID, groupID); err != nil {
		writePresetsError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func (h *PresetsHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	groups, err := h.presets.ListGroups(ctx, userID)
	if err != nil {
		writePresetsError(w, err)
		return
	}

	response := make([]groupResponse, 0, len(groups))
	for _, group := range groups {
		createdAt := group.CreatedAt
		updatedAt := group.UpdatedAt
		response = append(response, groupResponse{
			ID:        group.ID,
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
			UserID:    group.UserID.String(),
			Name:      group.Name,
			Public:    group.Public,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *PresetsHandler) ListGroupsWithPresets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	groups, err := h.presets.ListGroupsWithPresets(ctx, userID)
	if err != nil {
		writePresetsError(w, err)
		return
	}

	response := make([]groupWithPresetsResponse, 0, len(groups))
	for _, group := range groups {
		createdAt := group.CreatedAt
		updatedAt := group.UpdatedAt
		presetsOut := make([]presetInGroupResponse, 0, len(group.Presets))
		for _, p := range group.Presets {
			pc := p.CreatedAt
			pu := p.UpdatedAt
			presetsOut = append(presetsOut, presetInGroupResponse{
				ID:         p.ID,
				CreatedAt:  &pc,
				UpdatedAt:  &pu,
				GroupID:    p.GroupID,
				Type:       p.Type,
				Name:       p.Name,
				Public:     p.Public,
				AppVersion: p.AppVersion,
				Preset:     json.RawMessage(p.Preset),
			})
		}
		response = append(response, groupWithPresetsResponse{
			ID:        group.ID,
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
			UserID:    group.UserID.String(),
			Name:      group.Name,
			Public:    group.Public,
			Presets:   presetsOut,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *PresetsHandler) CreatePreset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	req, err := httpx.DecodeJSON[createPresetRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	result, err := h.presets.CreatePreset(ctx, userID, presets.CreatePresetInput{
		GroupID:    req.GroupID,
		Type:       req.Type,
		Name:       req.Name,
		Public:     req.Public,
		AppVersion: req.AppVersion,
		Preset:     datatypes.JSON(req.Preset),
	})
	if err != nil {
		writePresetsError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, presetResultResponse{
		ID: result.ID,
	})
}

func (h *PresetsHandler) ListPresets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	presetsList, err := h.presets.ListPresets(ctx, userID)
	if err != nil {
		writePresetsError(w, err)
		return
	}

	response := make([]presetResponse, 0, len(presetsList))
	for _, preset := range presetsList {
		createdAt := preset.CreatedAt
		updatedAt := preset.UpdatedAt
		response = append(response, presetResponse{
			ID:         preset.ID,
			CreatedAt:  &createdAt,
			UpdatedAt:  &updatedAt,
			GroupID:    preset.GroupID,
			GroupName:  preset.GroupName,
			Type:       preset.Type,
			Name:       preset.Name,
			Public:     preset.Public,
			AppVersion: preset.AppVersion,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *PresetsHandler) ListPresetsSharedWithUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	shared, err := h.presets.ListPresetsSharedWithUser(ctx, userID)
	if err != nil {
		writePresetsError(w, err)
		return
	}

	response := make([]sharedPresetsResponse, 0, len(shared))
	for _, block := range shared {
		groupsOut := make([]sharedPresetGroupResponse, 0, len(block.Groups))
		for _, grp := range block.Groups {
			gc := grp.CreatedAt
			gu := grp.UpdatedAt
			presetsOut := make([]presetInGroupResponse, 0, len(grp.Presets))
			for _, p := range grp.Presets {
				pc := p.CreatedAt
				pu := p.UpdatedAt
				presetsOut = append(presetsOut, presetInGroupResponse{
					ID:         p.ID,
					CreatedAt:  &pc,
					UpdatedAt:  &pu,
					GroupID:    p.GroupID,
					Type:       p.Type,
					Name:       p.Name,
					Public:     p.Public,
					AppVersion: p.AppVersion,
					Preset:     json.RawMessage(p.Preset),
				})
			}
			groupsOut = append(groupsOut, sharedPresetGroupResponse{
				ID:        grp.ID,
				CreatedAt: &gc,
				UpdatedAt: &gu,
				Name:      grp.Name,
				Public:    grp.Public,
				Presets:   presetsOut,
			})
		}
		response = append(response, sharedPresetsResponse{
			Owner: sharedPresetOwnerResponse{
				ID:        block.Owner.ID.String(),
				Username:  block.Owner.Username,
				FirstName: block.Owner.FirstName,
				LastName:  block.Owner.LastName,
			},
			Groups: groupsOut,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *PresetsHandler) ListPresetsInGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	groupID, ok := httpx.UintURLParam(w, r, "groupID")
	if !ok {
		return
	}

	presetsList, err := h.presets.ListPresetsInGroup(ctx, userID, groupID)
	if err != nil {
		writePresetsError(w, err)
		return
	}

	response := make([]presetResponse, 0, len(presetsList))
	for _, preset := range presetsList {
		createdAt := preset.CreatedAt
		updatedAt := preset.UpdatedAt
		response = append(response, presetResponse{
			ID:         preset.ID,
			CreatedAt:  &createdAt,
			UpdatedAt:  &updatedAt,
			GroupID:    preset.GroupID,
			GroupName:  preset.GroupName,
			Type:       preset.Type,
			Name:       preset.Name,
			Public:     preset.Public,
			AppVersion: preset.AppVersion,
			Preset:     json.RawMessage(preset.Preset),
		})
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *PresetsHandler) UpdatePreset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	presetID, ok := httpx.UintURLParam(w, r, "presetID")
	if !ok {
		return
	}

	req, err := httpx.DecodeJSON[updatePresetRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	var presetData *datatypes.JSON
	if req.Preset != nil {
		data := datatypes.JSON(*req.Preset)
		presetData = &data
	}

	result, err := h.presets.UpdatePreset(ctx, userID, presets.UpdatePresetInput{
		PresetID:   presetID,
		Type:       req.Type,
		Name:       req.Name,
		Public:     req.Public,
		AppVersion: req.AppVersion,
		Preset:     presetData,
	})
	if err != nil {
		writePresetsError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, presetResultResponse{
		ID: result.ID,
	})
}

func (h *PresetsHandler) DeletePreset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	presetID, ok := httpx.UintURLParam(w, r, "presetID")
	if !ok {
		return
	}

	if err := h.presets.DeletePreset(ctx, userID, presetID); err != nil {
		writePresetsError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func (h *PresetsHandler) SharePreset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	presetID, ok := httpx.UintURLParam(w, r, "presetID")
	if !ok {
		return
	}

	req, err := httpx.DecodeJSON[shareWithUserRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	result, err := h.presets.SharePreset(ctx, userID, presets.SharePresetInput{
		PresetID:        presetID,
		ShareWithUserID: req.ShareWithUserID,
	})
	if err != nil {
		writePresetsError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, shareRecordResponse{ID: result.ID})
}

func (h *PresetsHandler) ShareGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	groupID, ok := httpx.UintURLParam(w, r, "groupID")
	if !ok {
		return
	}

	req, err := httpx.DecodeJSON[shareWithUserRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	result, err := h.presets.ShareGroup(ctx, userID, presets.ShareGroupInput{
		GroupID:         groupID,
		ShareWithUserID: req.ShareWithUserID,
	})
	if err != nil {
		writePresetsError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, shareRecordResponse{ID: result.ID})
}

func writePresetsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, presets.ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, presets.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
	}
}
