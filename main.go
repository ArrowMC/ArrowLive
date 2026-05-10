package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"ArrowLiveWebsite/internal/api"
	"ArrowLiveWebsite/internal/auth"
	"ArrowLiveWebsite/internal/config"
	"ArrowLiveWebsite/internal/room"
	"ArrowLiveWebsite/internal/storage"
	"ArrowLiveWebsite/internal/webhook"
	"ArrowLiveWebsite/internal/zlmedia"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	store, err := storage.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer store.Close()

	if n, err := store.ResetOccupied(); err != nil {
		log.Fatalf("storage: reset occupied rooms: %v", err)
	} else if n > 0 {
		log.Printf("cleared %d occupied room(s) from previous run", n)
	}

	zlm := zlmedia.New(cfg.ZLMediaKit.APIBase, cfg.ZLMediaKit.Secret)
	rooms := room.New(store, &cfg.ZLMediaKit)
	authMgr := auth.NewManager(&cfg.Admin)

	userAPI := api.NewUserAPI(rooms)
	adminAPI := api.NewAdminAPI(rooms, zlm, authMgr, &cfg.ZLMediaKit)
	hookH := webhook.New(rooms, cfg.ZLMediaKit.App)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	// user-facing web + API
	r.Get("/", redirectTo("/user/"))
	r.Handle("/user/*", http.StripPrefix("/user/", http.FileServer(http.Dir("web/user"))))
	r.Route("/user/api", func(r chi.Router) {
		r.Post("/rooms", userAPI.CreateRoom)
		r.Get("/rooms/{name}/status", userAPI.GetStatus)
	})

	// admin web
	r.Get("/admin", redirectTo("/admin/"))
	r.Handle("/admin/*", http.StripPrefix("/admin/", http.FileServer(http.Dir("web/admin"))))

	// admin API
	r.Route("/admin/api", func(r chi.Router) {
		r.Post("/login", adminAPI.Login)
		r.Post("/logout", adminAPI.Logout)
		r.Group(func(r chi.Router) {
			r.Use(authMgr.Middleware)
			r.Get("/rooms", adminAPI.ListRooms)
			r.Get("/groups", adminAPI.ListGroups)
		})
	})

	// zlmediakit webhook
	r.Route("/webhook", func(r chi.Router) {
		r.Post("/on_publish", hookH.OnPublish)
		r.Post("/on_stream_changed", hookH.OnStreamChanged)
		r.Post("/on_play", hookH.OnPlay)
		r.Post("/on_stream_none_reader", hookH.OnStreamNoneReader)
		r.Post("/on_stream_not_found", hookH.OnStreamNotFound)
		r.Post("/on_server_started", hookH.OnGeneric)
		r.Post("/on_server_keepalive", hookH.OnGeneric)
		r.Post("/on_flow_report", hookH.OnGeneric)
		r.Post("/on_record_mp4", hookH.OnGeneric)
		r.Post("/on_rtsp_auth", hookH.OnGeneric)
		r.Post("/on_rtsp_realm", hookH.OnGeneric)
		r.Post("/on_shell_login", hookH.OnGeneric)
		r.Post("/on_http_access", hookH.OnGeneric)
		r.Post("/on_rtp_server_timeout", hookH.OnGeneric)
	})

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")
	_ = srv.Close()
}

func redirectTo(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	}
}
