package jellyfin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/service"
	"github.com/gavinmcnair/tvproxy/pkg/store"
	"github.com/gavinmcnair/tvproxy/pkg/tmdb"
)

type Server struct {
	serverID     string
	serverName   string
	baseURL      string
	auth         *service.AuthService
	channels     store.ChannelStore
	streams      store.StreamReader
	epg          store.EPGStore
	logoService  *service.LogoService
	tmdbClient   *tmdb.Client
	log          zerolog.Logger
	tokens       sync.Map
}

func NewServer(serverName, baseURL string, auth *service.AuthService, channels store.ChannelStore, streams store.StreamReader, epg store.EPGStore, logoService *service.LogoService, tmdbClient *tmdb.Client, log zerolog.Logger) *Server {
	id := make([]byte, 16)
	rand.Read(id)
	h := hex.EncodeToString(id)
	guid := h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
	return &Server{
		serverID:    guid,
		serverName:  serverName,
		baseURL:     baseURL,
		auth:        auth,
		channels:    channels,
		streams:     streams,
		epg:         epg,
		logoService: logoService,
		tmdbClient:  tmdbClient,
		log:         log.With().Str("component", "jellyfin").Logger(),
	}
}

func (s *Server) Router() chi.Router {
	r := chi.NewRouter()

	r.Get("/System/Info/Public", s.systemInfoPublic)
	r.Get("/System/Info", s.systemInfo)
	r.Get("/System/Ping", s.ping)
	r.Post("/System/Ping", s.ping)

	r.Get("/Branding/Configuration", s.brandingConfig)
	r.Get("/Branding/Css", s.brandingCSS)

	r.Get("/QuickConnect/Enabled", s.quickConnectEnabled)

	r.Get("/Users/Public", s.usersPublic)
	r.Post("/Users/AuthenticateByName", s.authenticateByName)

	r.Group(func(r chi.Router) {
		r.Use(s.requireAuth)

		r.Get("/Users/Me", s.usersMe)
		r.Get("/Users", s.usersList)
		r.Get("/Users/{userId}", s.userByID)

		r.Get("/UserViews", s.userViews)

		r.Get("/Items", s.getItems)
		r.Get("/Items/{itemId}", s.getItem)
		r.Get("/Items/Latest", s.getLatest)
		r.Get("/Items/Resume", s.getResume)
		r.Get("/UserItems/Resume", s.getResume)

		r.Get("/Shows/{seriesId}/Seasons", s.getSeasons)
		r.Get("/Shows/{seriesId}/Episodes", s.getEpisodes)

		r.Get("/Items/{itemId}/Images/{imageType}", s.getImage)
		r.Get("/Items/{itemId}/Images/{imageType}/{imageIndex}", s.getImage)
		r.Head("/Items/{itemId}/Images/{imageType}", s.getImage)
		r.Head("/Items/{itemId}/Images/{imageType}/{imageIndex}", s.getImage)

		r.Post("/Items/{itemId}/PlaybackInfo", s.playbackInfo)
		r.Get("/Videos/{itemId}/stream", s.videoStream)
		r.Get("/Videos/{itemId}/stream.{container}", s.videoStream)
		r.Head("/Videos/{itemId}/stream", s.videoStream)
		r.Head("/Videos/{itemId}/stream.{container}", s.videoStream)

		r.Get("/LiveTv/Info", s.liveTvInfo)
		r.Get("/LiveTv/Channels", s.liveTvChannels)
		r.Get("/LiveTv/Programs", s.liveTvPrograms)
		r.Post("/LiveTv/Programs", s.liveTvPrograms)
		r.Get("/LiveTv/GuideInfo", s.liveTvGuideInfo)

		r.Post("/Sessions/Capabilities/Full", s.sessionsCapabilities)
		r.Post("/Sessions/Playing", s.sessionsPlaying)
		r.Post("/Sessions/Playing/Progress", s.sessionsPlaying)
		r.Post("/Sessions/Playing/Stopped", s.sessionsPlaying)

		r.Get("/DisplayPreferences/{id}", s.displayPreferences)

		r.Post("/UserPlayedItems/{itemId}", s.markPlayed)
		r.Delete("/UserPlayedItems/{itemId}", s.markPlayed)
		r.Post("/UserFavoriteItems/{itemId}", s.markFavorite)
		r.Delete("/UserFavoriteItems/{itemId}", s.markFavorite)
	})

	return r
}

func (s *Server) respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8; profile=\"CamelCase\"")
	w.Header().Set("X-Application", "Jellyfin")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) systemInfoPublic(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, PublicSystemInfo{
		LocalAddress:           s.baseURL,
		ServerName:             s.serverName,
		Version:                "10.10.6",
		ProductName:            "Jellyfin Server",
		OperatingSystem:        "Linux",
		ID:                     s.serverID,
		StartupWizardCompleted: true,
	})
}

func (s *Server) systemInfo(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]any{
		"LocalAddress":             s.baseURL,
		"ServerName":               s.serverName,
		"Version":                  "10.10.6",
		"ProductName":              "Jellyfin Server",
		"OperatingSystem":          "Linux",
		"OperatingSystemDisplayName": "Linux",
		"Id":                       s.serverID,
		"StartupWizardCompleted":   true,
		"HasPendingRestart":        false,
		"IsShuttingDown":           false,
		"SupportsLibraryMonitor":   false,
		"WebSocketPortNumber":      0,
		"CanSelfRestart":           false,
		"CanLaunchWebBrowser":      false,
		"HasUpdateAvailable":       false,
	})
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Jellyfin Server"))
}

func (s *Server) brandingConfig(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, BrandingConfiguration{
		LoginDisclaimer:     "",
		CustomCSS:           "",
		SplashscreenEnabled: false,
	})
}

func (s *Server) brandingCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	w.Write([]byte(""))
}

func (s *Server) quickConnectEnabled(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, false)
}

func (s *Server) usersPublic(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, []UserDto{
		{
			Name:        "admin",
			ID:          "00000000000000000000000000000001",
			HasPassword: true,
			HasConfiguredPassword: true,
		},
	})
}

func (s *Server) authenticateByName(w http.ResponseWriter, r *http.Request) {
	var req AuthenticateByNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	_, _, err := s.auth.Login(r.Context(), req.Username, req.Pw)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	users, _ := s.auth.ListUsers(r.Context())
	var userID, userName string
	var isAdmin bool
	for _, u := range users {
		if strings.EqualFold(u.Username, req.Username) {
			userID = strings.ReplaceAll(u.ID, "-", "")
			userName = u.Username
			isAdmin = u.IsAdmin
			break
		}
	}

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)
	s.tokens.Store(token, userID)

	now := time.Now()
	s.respondJSON(w, http.StatusOK, AuthenticationResult{
		User: &UserDto{
			Name:                  userName,
			ServerID:              s.serverID,
			ServerName:            s.serverName,
			ID:                    userID,
			HasPassword:           true,
			HasConfiguredPassword: true,
			LastLoginDate:         &now,
			LastActivityDate:      &now,
			Configuration: UserConfig{
				PlayDefaultAudioTrack: true,
				SubtitleMode:          "Default",
			},
			Policy: s.defaultPolicy(isAdmin),
		},
		SessionInfo: &SessionInfo{
			ID:                 token[:16],
			UserID:             userID,
			UserName:           userName,
			Client:             s.extractClient(r),
			LastActivityDate:   now,
			DeviceName:         s.extractDevice(r),
			DeviceID:           s.extractDeviceID(r),
			ApplicationVersion: s.extractVersion(r),
			IsActive:           true,
			ServerID:           s.serverID,
		},
		AccessToken: token,
		ServerID:    s.serverID,
	})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := s.extractToken(r)
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if _, ok := s.tokens.Load(token); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) extractToken(r *http.Request) string {
	if t := r.URL.Query().Get("api_key"); t != "" {
		return t
	}
	if t := r.URL.Query().Get("ApiKey"); t != "" {
		return t
	}
	if t := r.Header.Get("X-MediaBrowser-Token"); t != "" {
		return t
	}
	if t := r.Header.Get("X-Emby-Token"); t != "" {
		return t
	}
	auth := r.Header.Get("Authorization")
	if auth == "" {
		auth = r.Header.Get("X-Emby-Authorization")
	}
	if strings.Contains(auth, "Token=") {
		parts := strings.Split(auth, "Token=")
		if len(parts) > 1 {
			token := strings.Trim(parts[1], "\" ,")
			return token
		}
	}
	return ""
}

func (s *Server) extractFromAuth(r *http.Request, key string) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		auth = r.Header.Get("X-Emby-Authorization")
	}
	if !strings.Contains(auth, key+"=") {
		return ""
	}
	parts := strings.Split(auth, key+"=")
	if len(parts) < 2 {
		return ""
	}
	val := parts[1]
	if strings.HasPrefix(val, "\"") {
		end := strings.Index(val[1:], "\"")
		if end >= 0 {
			return val[1 : end+1]
		}
	}
	end := strings.IndexAny(val, ", ")
	if end >= 0 {
		return val[:end]
	}
	return val
}

func (s *Server) extractClient(r *http.Request) string {
	return s.extractFromAuth(r, "Client")
}

func (s *Server) extractDevice(r *http.Request) string {
	return s.extractFromAuth(r, "Device")
}

func (s *Server) extractDeviceID(r *http.Request) string {
	return s.extractFromAuth(r, "DeviceId")
}

func (s *Server) extractVersion(r *http.Request) string {
	return s.extractFromAuth(r, "Version")
}

func (s *Server) getUserID(r *http.Request) string {
	token := s.extractToken(r)
	if v, ok := s.tokens.Load(token); ok {
		return v.(string)
	}
	return ""
}

func (s *Server) defaultPolicy(isAdmin bool) UserPolicy {
	return UserPolicy{
		IsAdministrator:                isAdmin,
		IsDisabled:                     false,
		EnableUserPreferenceAccess:     true,
		EnableRemoteControlOfOtherUsers: isAdmin,
		EnableSharedDeviceControl:      true,
		EnableRemoteAccess:             true,
		EnableLiveTvManagement:         isAdmin,
		EnableLiveTvAccess:             true,
		EnableMediaPlayback:            true,
		EnableAudioPlaybackTranscoding: true,
		EnableVideoPlaybackTranscoding: true,
		EnablePlaybackRemuxing:         true,
		EnableContentDownloading:       true,
		EnableAllChannels:              true,
		EnableAllFolders:               true,
		EnableAllDevices:               true,
		EnablePublicSharing:            true,
		AuthenticationProviderId:       "Jellyfin.Server.Implementations.Users.DefaultAuthenticationProvider",
		PasswordResetProviderId:        "Jellyfin.Server.Implementations.Users.DefaultPasswordResetProvider",
	}
}

func (s *Server) usersMe(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserID(r)
	now := time.Now()
	s.respondJSON(w, http.StatusOK, UserDto{
		Name:                  "admin",
		ServerID:              s.serverID,
		ServerName:            s.serverName,
		ID:                    userID,
		HasPassword:           true,
		HasConfiguredPassword: true,
		LastLoginDate:         &now,
		LastActivityDate:      &now,
		Configuration: UserConfig{
			PlayDefaultAudioTrack: true,
			SubtitleMode:          "Default",
		},
		Policy: s.defaultPolicy(true),
	})
}

func (s *Server) usersList(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserID(r)
	now := time.Now()
	s.respondJSON(w, http.StatusOK, []UserDto{
		{
			Name:                  "admin",
			ServerID:              s.serverID,
			ServerName:            s.serverName,
			ID:                    userID,
			HasPassword:           true,
			HasConfiguredPassword: true,
			LastLoginDate:         &now,
			LastActivityDate:      &now,
			Configuration: UserConfig{
				PlayDefaultAudioTrack: true,
				SubtitleMode:          "Default",
			},
			Policy: s.defaultPolicy(true),
		},
	})
}

func (s *Server) userByID(w http.ResponseWriter, r *http.Request) {
	s.usersMe(w, r)
}

func (s *Server) sessionsCapabilities(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) sessionsPlaying(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) displayPreferences(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]any{
		"Id":              chi.URLParam(r, "id"),
		"SortBy":          "SortName",
		"SortOrder":       "Ascending",
		"RememberIndexing": false,
		"RememberSorting":  false,
		"Client":          s.extractClient(r),
		"CustomPrefs":     map[string]string{},
	})
}

func (s *Server) markPlayed(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, UserItemData{
		Played: r.Method == "POST",
		Key:    chi.URLParam(r, "itemId"),
	})
}

func (s *Server) markFavorite(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, UserItemData{
		IsFavorite: r.Method == "POST",
		Key:        chi.URLParam(r, "itemId"),
	})
}
