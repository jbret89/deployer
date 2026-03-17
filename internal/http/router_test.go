package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"deployer/internal/config"
	"deployer/internal/service"
)

type fakeDeployer struct {
	deployLogs       string
	deployErr        error
	rollbackLogs     string
	rollbackErr      error
	deployCalledWith string
	rollCalledWith   string
}

func (f *fakeDeployer) Deploy(_ context.Context, repo string) (string, error) {
	f.deployCalledWith = repo
	return f.deployLogs, f.deployErr
}

func (f *fakeDeployer) Rollback(_ context.Context, repo string) (string, error) {
	f.rollCalledWith = repo
	return f.rollbackLogs, f.rollbackErr
}

func TestDeployHandlerRejectsInvalidToken(t *testing.T) {
	handler := buildTestHandler(config.Config{AdminToken: "secret"}, &fakeDeployer{})

	req := httptest.NewRequest(http.MethodPost, "/deploy/app", nil)
	req.SetPathValue("repo", "app")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestDeployHandlerRejectsInvalidRepoName(t *testing.T) {
	fake := &fakeDeployer{}
	handler := buildTestHandler(config.Config{AdminToken: "secret"}, fake)

	req := httptest.NewRequest(http.MethodPost, "/deploy/app.name", nil)
	req.Header.Set("X-Admin-Token", "secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if fake.deployCalledWith != "" {
		t.Fatalf("expected deployer not to be called, got %q", fake.deployCalledWith)
	}
}

func TestDeployHandlerReturnsLogsOnSuccess(t *testing.T) {
	fake := &fakeDeployer{deployLogs: "deploy ok"}
	handler := buildTestHandler(config.Config{AdminToken: "secret"}, fake)

	req := httptest.NewRequest(http.MethodPost, "/deploy/app", nil)
	req.Header.Set("X-Admin-Token", "secret")
	req.SetPathValue("repo", "app")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["repo"] != "app" {
		t.Fatalf("expected repo app, got %q", payload["repo"])
	}
	if payload["logs"] != "deploy ok" {
		t.Fatalf("expected logs to be returned, got %q", payload["logs"])
	}
}

func TestRollbackHandlerReturnsLogsOnSuccess(t *testing.T) {
	fake := &fakeDeployer{rollbackLogs: "rollback ok"}
	handler := buildTestHandler(config.Config{AdminToken: "secret"}, fake)

	req := httptest.NewRequest(http.MethodPost, "/rollback/app", nil)
	req.Header.Set("X-Admin-Token", "secret")
	req.SetPathValue("repo", "app")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["logs"] != "rollback ok" {
		t.Fatalf("expected rollback logs, got %q", payload["logs"])
	}
}

func TestDeployHandlerSanitizesResponseFields(t *testing.T) {
	fake := &fakeDeployer{deployLogs: "ok\x00\r\nnext"}
	handler := buildTestHandler(config.Config{AdminToken: "secret"}, fake)

	req := httptest.NewRequest(http.MethodPost, "/deploy/app", nil)
	req.Header.Set("X-Admin-Token", "secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["logs"] != "ok\nnext" {
		t.Fatalf("expected sanitized logs, got %q", payload["logs"])
	}
}

func TestDeployHandlerReturnsConflictWhenRepoIsBusy(t *testing.T) {
	fake := &fakeDeployer{deployErr: service.ErrRepoBusy}
	handler := buildTestHandler(config.Config{AdminToken: "secret"}, fake)

	req := httptest.NewRequest(http.MethodPost, "/deploy/app", nil)
	req.Header.Set("X-Admin-Token", "secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !errors.Is(fake.deployErr, service.ErrRepoBusy) {
		t.Fatal("expected busy error to be preserved")
	}
	if payload["error"] != service.ErrRepoBusy.Error() {
		t.Fatalf("expected busy error message, got %q", payload["error"])
	}
}

func buildTestHandler(cfg config.Config, deployer deployerAPI) http.Handler {
	mux := http.NewServeMux()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registerRoutes(mux, cfg, logger, deployer)
	return mux
}
