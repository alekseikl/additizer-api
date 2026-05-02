package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/alekseikl/additizer-api/internal/httpx"
	"github.com/alekseikl/additizer-api/internal/middleware"
	"github.com/alekseikl/additizer-api/internal/models"
	"github.com/alekseikl/additizer-api/internal/presets"
	"github.com/go-chi/chi/v5"
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

	groupID, ok := uintURLParam(w, r, "groupID")
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

	groupID, ok := uintURLParam(w, r, "groupID")
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

func (h *PresetsHandler) ListPresetsInGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	groupID, ok := uintURLParam(w, r, "groupID")
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

	presetID, ok := uintURLParam(w, r, "presetID")
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

	presetID, ok := uintURLParam(w, r, "presetID")
	if !ok {
		return
	}

	if err := h.presets.DeletePreset(ctx, userID, presetID); err != nil {
		writePresetsError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func uintURLParam(w http.ResponseWriter, r *http.Request, name string) (uint, bool) {
	value, err := strconv.ParseUint(chi.URLParam(r, name), 10, 0)
	if err != nil || value == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return uint(value), true
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
