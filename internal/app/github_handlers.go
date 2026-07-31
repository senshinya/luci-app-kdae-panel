package app

import (
	"errors"
	"net/http"

	"github.com/tuoro/kdae-panel/internal/githubauth"
)

type GitHubCredentialService interface {
	Status() githubauth.Status
	SetToken(token string) error
	ClearToken() error
}

type githubTokenRequest struct {
	Token string `json:"token"`
}

func registerGitHubCredentialRoutes(router *http.ServeMux, service GitHubCredentialService) {
	unavailable := func(writer http.ResponseWriter) {
		writeAPIError(writer, http.StatusServiceUnavailable, "github_credentials_unavailable",
			"GitHub API 凭据管理不可用")
	}

	router.HandleFunc("GET /api/v1/settings/github", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		if service == nil {
			unavailable(writer)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"status": service.Status()})
	})

	router.HandleFunc("PUT /api/v1/settings/github", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		if service == nil {
			unavailable(writer)
			return
		}
		var payload githubTokenRequest
		if !decodeSmallJSONBody(writer, request, &payload) {
			return
		}
		if err := service.SetToken(payload.Token); err != nil {
			writeGitHubCredentialError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"status": service.Status()})
	})

	router.HandleFunc("DELETE /api/v1/settings/github", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		if service == nil {
			unavailable(writer)
			return
		}
		if err := service.ClearToken(); err != nil {
			writeGitHubCredentialError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"status": service.Status()})
	})
}

func writeGitHubCredentialError(writer http.ResponseWriter, err error) {
	if errors.Is(err, githubauth.ErrEnvironmentManaged) {
		writeAPIError(writer, http.StatusConflict, "github_token_environment_managed", err.Error())
		return
	}
	if errors.Is(err, githubauth.ErrInvalidToken) {
		writeAPIError(writer, http.StatusBadRequest, "github_token_invalid", err.Error())
		return
	}
	writeAPIError(writer, http.StatusInternalServerError, "github_token_save_failed", err.Error())
}
