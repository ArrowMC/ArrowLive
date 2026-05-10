package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"ArrowLiveWebsite/internal/auth"
	"ArrowLiveWebsite/internal/config"
	"ArrowLiveWebsite/internal/room"
	"ArrowLiveWebsite/internal/zlmedia"
)

type AdminAPI struct {
	rooms *room.Service
	zlm   *zlmedia.Client
	auth  *auth.Manager
	zcfg  *config.ZLMediaKit
}

func NewAdminAPI(rooms *room.Service, zlm *zlmedia.Client, am *auth.Manager, zcfg *config.ZLMediaKit) *AdminAPI {
	return &AdminAPI{rooms: rooms, zlm: zlm, auth: am, zcfg: zcfg}
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *AdminAPI) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp{"invalid body"})
		return
	}
	if !a.auth.Check(req.Username, req.Password) {
		writeJSON(w, http.StatusUnauthorized, errResp{"invalid credentials"})
		return
	}
	if err := a.auth.Issue(w); err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp{err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *AdminAPI) Logout(w http.ResponseWriter, r *http.Request) {
	a.auth.Revoke(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type roomView struct {
	Name    string `json:"name"`
	PlayURL string `json:"play_url"`
}

// activeRoomNames 以 ZLMediaKit getMediaList 为权威，返回指定 app 下的所有活动流名。
// 同一 stream 在多种 schema（rtmp/flv/rtsp 等）下会出现多条，这里按 stream 去重并保序。
// 不再与 sqlite 做交集：webhook 可能丢失、流可能由外部工具推送，都会让交集漏掉真实存在的流。
func (a *AdminAPI) activeRoomNames() ([]string, error) {
	items, err := a.zlm.GetMediaList(a.zcfg.App, "")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it.Stream == "" {
			continue
		}
		if _, ok := seen[it.Stream]; ok {
			continue
		}
		seen[it.Stream] = struct{}{}
		out = append(out, it.Stream)
	}
	return out, nil
}

func (a *AdminAPI) toViews(names []string) []roomView {
	out := make([]roomView, 0, len(names))
	for _, n := range names {
		out = append(out, roomView{
			Name:    n,
			PlayURL: fmt.Sprintf("%s/%s/%s.live.ts", a.zcfg.HTTPFLVPlayBase, a.zcfg.App, n),
		})
	}
	return out
}

func (a *AdminAPI) ListRooms(w http.ResponseWriter, r *http.Request) {
	names, err := a.activeRoomNames()
	if err != nil {
		log.Printf("admin: list rooms: %v", err)
		writeJSON(w, http.StatusBadGateway, errResp{"upstream error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rooms": a.toViews(names)})
}

func (a *AdminAPI) ListGroups(w http.ResponseWriter, r *http.Request) {
	names, err := a.activeRoomNames()
	if err != nil {
		log.Printf("admin: list groups: %v", err)
		writeJSON(w, http.StatusBadGateway, errResp{"upstream error"})
		return
	}
	views := a.toViews(names)
	groups := make([][]roomView, 0)
	for i := 0; i < len(views); i += 4 {
		end := i + 4
		if end > len(views) {
			end = len(views)
		}
		groups = append(groups, views[i:end])
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": groups})
}
