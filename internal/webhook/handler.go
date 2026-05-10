package webhook

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"ArrowLiveWebsite/internal/room"
	"ArrowLiveWebsite/internal/storage"
)

type Handler struct {
	rooms *room.Service
	app   string
}

func New(rooms *room.Service, app string) *Handler {
	return &Handler{rooms: rooms, app: app}
}

type okResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func writeOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(okResp{Code: 0, Msg: "success"})
}

func writeDeny(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(okResp{Code: -1, Msg: msg})
}

type publishReq struct {
	App    string `json:"app"`
	Stream string `json:"stream"`
	Params string `json:"params"`
	Schema string `json:"schema"`
	Vhost  string `json:"vhost"`
}

type publishResp struct {
	Code         int    `json:"code"`
	Msg          string `json:"msg"`
	EnableRTSP   bool   `json:"enable_rtsp"`
	EnableRTMP   bool   `json:"enable_rtmp"`
	EnableTS     bool   `json:"enable_ts"`
	EnableFMP4   bool   `json:"enable_fmp4"`
	EnableHLS    bool   `json:"enable_hls"`
	EnableMP4    bool   `json:"enable_mp4"`
	EnableAudio  bool   `json:"enable_audio"`
	AddMuteAudio bool   `json:"add_mute_audio"`
}

func (h *Handler) OnPublish(w http.ResponseWriter, r *http.Request) {
	var req publishReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeDeny(w, "invalid payload")
		return
	}
	if h.app != "" && req.App != h.app {
		writeDeny(w, "app not allowed")
		return
	}
	token := parseToken(req.Params)
	if token == "" {
		writeDeny(w, "token required")
		return
	}
	rm, err := h.rooms.Get(req.Stream)
	if err != nil {
		writeDeny(w, "room not found")
		return
	}
	token = strings.Replace(token, "/", "", 1)
	if rm.Token != token {
		writeDeny(w, "token mismatch")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(publishResp{
		Code:         0,
		Msg:          "success",
		EnableRTSP:   true,
		EnableRTMP:   true,
		EnableTS:     true,
		EnableFMP4:   false,
		EnableHLS:    false,
		EnableMP4:    false,
		EnableAudio:  true,
		AddMuteAudio: true,
	})
}

type streamChangedReq struct {
	App    string `json:"app"`
	Stream string `json:"stream"`
	Schema string `json:"schema"`
	Regist bool   `json:"regist"`
}

func (h *Handler) OnStreamChanged(w http.ResponseWriter, r *http.Request) {
	var req streamChangedReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOK(w)
		return
	}
	if h.app != "" && req.App != h.app {
		writeOK(w)
		return
	}
	if _, err := h.rooms.Get(req.Stream); err != nil {
		writeOK(w)
		return
	}
	target := storage.StatusInactive
	if req.Regist {
		target = storage.StatusActive
	}
	if err := h.rooms.SetStatus(req.Stream, target); err != nil {
		log.Printf("webhook: set status %s -> %s failed: %v", req.Stream, target, err)
	}
	writeOK(w)
}

func (h *Handler) OnPlay(w http.ResponseWriter, r *http.Request) {
	writeOK(w)
}

type noneReaderResp struct {
	Code  int  `json:"code"`
	Close bool `json:"close"`
}

func (h *Handler) OnStreamNoneReader(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(noneReaderResp{Code: 0, Close: false})
}

type notFoundReq struct {
	App    string `json:"app"`
	Stream string `json:"stream"`
	Schema string `json:"schema"`
}

func (h *Handler) OnStreamNotFound(w http.ResponseWriter, r *http.Request) {
	var req notFoundReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	log.Printf("webhook: stream not found app=%s stream=%s schema=%s", req.App, req.Stream, req.Schema)
	writeOK(w)
}

func (h *Handler) OnGeneric(w http.ResponseWriter, r *http.Request) {
	writeOK(w)
}

func parseToken(params string) string {
	params = strings.TrimSpace(params)
	if params == "" {
		return ""
	}
	v, err := url.ParseQuery(params)
	if err != nil {
		return ""
	}
	return v.Get("token")
}
