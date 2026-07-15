package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrey-vk/wdsplit/internal/settings"
)

type settingDTO struct {
	Key       string   `json:"key"`
	Value     string   `json:"value,omitempty"`
	Default   string   `json:"default,omitempty"`
	EnvVar    string   `json:"env_var,omitempty"`
	EnvLocked bool     `json:"env_locked"`
	Editable  bool     `json:"editable"`
	Secret    bool     `json:"secret"`
	Type      string   `json:"type"`
	Options   []string `json:"options,omitempty"`
}

// handleListSettings handles GET /api/admin/settings.
func (s *Server) handleListSettings(w http.ResponseWriter, _ *http.Request) {
	list := s.settings.List()
	dtos := make([]settingDTO, 0, len(list))
	for _, info := range list {
		dtos = append(dtos, settingDTO{
			Key:       info.Key(),
			Value:     info.ValueString(),
			Default:   info.DefaultString(),
			EnvVar:    info.EnvVar(),
			EnvLocked: info.IsEnvLocked(),
			Editable:  info.Editable(),
			Secret:    info.IsSecret(),
			Type:      info.UIType(),
			Options:   info.Options(),
		})
	}
	writeJSON(w, http.StatusOK, dtos)
}

// handleUpdateSettings handles PUT /api/admin/settings. The body is
// {key: rawValue}; every key is validated before any is applied, so a
// batch either fully succeeds or changes nothing.
func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "Invalid request body"})
		return
	}

	byKey := make(map[string]settings.Info, len(s.settings.List()))
	for _, info := range s.settings.List() {
		byKey[info.Key()] = info
	}

	for key := range body {
		if _, ok := byKey[key]; !ok {
			writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: fmt.Sprintf("unknown setting %q", key)})
			return
		}
	}
	for key, raw := range body {
		if err := byKey[key].Validate(raw); err != nil {
			writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: fmt.Sprintf("%s: %v", key, err)})
			return
		}
	}

	for key, raw := range body {
		if err := byKey[key].Set(r.Context(), raw); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: fmt.Sprintf("%s: %v", key, err)})
			return
		}
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}
