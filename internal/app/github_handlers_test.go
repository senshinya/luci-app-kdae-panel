package app

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tuoro/kdae-panel/internal/githubauth"
)

type stubGitHubCredentials struct {
	status githubauth.Status
	token  string
	err    error
}

func (s *stubGitHubCredentials) Status() githubauth.Status { return s.status }

func (s *stubGitHubCredentials) SetToken(token string) error {
	if s.err != nil {
		return s.err
	}
	s.token = token
	s.status = githubauth.Status{Configured: true, Source: "panel"}
	return nil
}

func (s *stubGitHubCredentials) ClearToken() error {
	if s.err != nil {
		return s.err
	}
	s.token = ""
	s.status = githubauth.Status{}
	return nil
}

func newGitHubSettingsApp(t *testing.T, service GitHubCredentialService) *App {
	t.Helper()
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, GitHub: service})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close() })
	return application
}

func TestGitHubCredentialRoutesNeverReturnToken(t *testing.T) {
	service := &stubGitHubCredentials{}
	application := newGitHubSettingsApp(t, service)
	const token = "github_pat_0123456789abcdefghijklmnop"

	put := httptest.NewRecorder()
	application.Handler().ServeHTTP(put, httptest.NewRequest(http.MethodPut,
		"/api/v1/settings/github", strings.NewReader(`{"token":"`+token+`"}`)))
	if put.Code != http.StatusOK || !strings.Contains(put.Body.String(), `"configured":true`) {
		t.Fatalf("保存响应 = %d %s", put.Code, put.Body.String())
	}
	if strings.Contains(put.Body.String(), token) {
		t.Fatal("保存响应泄露了 Token")
	}

	get := httptest.NewRecorder()
	application.Handler().ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/settings/github", nil))
	if get.Code != http.StatusOK || strings.Contains(get.Body.String(), token) {
		t.Fatalf("读取响应 = %d %s", get.Code, get.Body.String())
	}
	if get.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", get.Header().Get("Cache-Control"))
	}

	deleteResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(deleteResponse,
		httptest.NewRequest(http.MethodDelete, "/api/v1/settings/github", nil))
	if deleteResponse.Code != http.StatusOK || service.token != "" {
		t.Fatalf("清除响应 = %d %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestGitHubCredentialRoutesPreserveErrorSemantics(t *testing.T) {
	for _, item := range []struct {
		err  error
		code int
	}{
		{githubauth.ErrInvalidToken, http.StatusBadRequest},
		{githubauth.ErrEnvironmentManaged, http.StatusConflict},
		{errors.New("disk full"), http.StatusInternalServerError},
	} {
		service := &stubGitHubCredentials{err: item.err}
		application := newGitHubSettingsApp(t, service)
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPut,
			"/api/v1/settings/github", strings.NewReader(`{"token":"github_pat_0123456789abcdefghijklmnop"}`)))
		if response.Code != item.code {
			t.Fatalf("错误 %v 的响应 = %d %s，期望 %d", item.err, response.Code, response.Body.String(), item.code)
		}
	}
}
