package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ArrowLiveWebsite/internal/room"
	"ArrowLiveWebsite/internal/storage"
)

type UserAPI struct {
	rooms *room.Service
}

func NewUserAPI(rooms *room.Service) *UserAPI {
	return &UserAPI{rooms: rooms}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type createReq struct {
	Name string `json:"name"`
}

type errResp struct {
	Error string `json:"error"`
}

func (u *UserAPI) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp{"invalid body"})
		return
	}
	info, err := u.rooms.Create(req.Name)
	if err != nil {
		switch {
		case errors.Is(err, room.ErrInvalidName):
			writeJSON(w, http.StatusBadRequest, errResp{"invalid name: 1-32 chars, [a-zA-Z0-9_-] only"})
		case errors.Is(err, room.ErrOccupied):
			writeJSON(w, http.StatusConflict, errResp{"room is currently occupied"})
		default:
			writeJSON(w, http.StatusInternalServerError, errResp{err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (u *UserAPI) GetStatus(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := room.ValidateName(name); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp{"invalid name"})
		return
	}
	info, err := u.rooms.Status(name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errResp{"not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errResp{err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":   info.Name,
		"status": info.Status,
		"active": info.Status == storage.StatusActive,
	})
}
