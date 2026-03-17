package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"deployer/internal/config"
	"deployer/internal/service"
)

type Server struct {
	httpServer *http.Server
}

type deployerAPI interface {
	Deploy(ctx context.Context, repo string) (string, error)
	Rollback(ctx context.Context, repo string) (string, error)
}

type appHandler struct {
	logger   *slog.Logger
	deployer deployerAPI
}

func NewServer(cfg config.Config, logger *slog.Logger, deployer *service.Deployer) *Server {
	mux := http.NewServeMux()
	registerRoutes(mux, cfg, logger, deployer)

	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Address(),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func registerRoutes(mux *http.ServeMux, cfg config.Config, logger *slog.Logger, deployer deployerAPI) {
	handler := appHandler{
		logger:   logger,
		deployer: deployer,
	}

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	mux.Handle("POST /deploy/{repo}", adminTokenMiddleware(cfg, http.HandlerFunc(handler.handleDeploy)))
	mux.Handle("POST /rollback/{repo}", adminTokenMiddleware(cfg, http.HandlerFunc(handler.handleRollback)))
}

func (h appHandler) handleDeploy(w http.ResponseWriter, r *http.Request) {
	repo, ok := repoFromRequest(w, r)
	if !ok {
		return
	}

	h.logger.Info("deploy requested", "repo", repo)
	logs, err := h.deployer.Deploy(r.Context(), repo)
	if err != nil {
		h.logger.Error("deploy failed", "repo", repo, "error", err)
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrRepoBusy) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{
			"error": sanitizeResponseValue(err.Error()),
			"logs":  sanitizeResponseValue(logs),
			"repo":  repo,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"repo": repo,
		"logs": sanitizeResponseValue(logs),
	})
}

func (h appHandler) handleRollback(w http.ResponseWriter, r *http.Request) {
	repo, ok := repoFromRequest(w, r)
	if !ok {
		return
	}

	h.logger.Info("rollback requested", "repo", repo)
	logs, err := h.deployer.Rollback(r.Context(), repo)
	if err != nil {
		h.logger.Error("rollback failed", "repo", repo, "error", err)
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrRepoBusy) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{
			"error": sanitizeResponseValue(err.Error()),
			"logs":  sanitizeResponseValue(logs),
			"repo":  repo,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"repo": repo,
		"logs": sanitizeResponseValue(logs),
	})
}

func repoFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	repo := r.PathValue("repo")
	if !isValidRepoName(repo) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid repo name",
		})
		return "", false
	}

	return repo, true
}

func adminTokenMiddleware(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providedToken := r.Header.Get("X-Admin-Token")
		expectedToken := cfg.AdminToken

		if expectedToken == "" || subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}

func sanitizeResponseValue(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 32 || r == 127:
			return -1
		default:
			return r
		}
	}, value)

	return strings.TrimSpace(value)
}
