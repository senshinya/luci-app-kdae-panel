package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/auth"
	"github.com/tuoro/kdae-panel/internal/configstore"
	"github.com/tuoro/kdae-panel/internal/daeconfig"
)

type groupPolicyService struct {
	stubConfigurationService
	gotGroup  string
	gotPolicy string
	gotHash   string
	err       error
}

func (s *groupPolicyService) SetGroupPolicy(_ context.Context, group, policy, expectedHash string) (configstore.SaveResult, error) {
	s.gotGroup, s.gotPolicy, s.gotHash = group, policy, expectedHash
	if s.err != nil {
		return configstore.SaveResult{}, s.err
	}
	return configstore.SaveResult{Hash: "新哈希", Applied: true}, nil
}

// newAuthenticatedTestHandler 建一个挂了认证中间件的应用并回一份可用会话。
//
// app_test.go 里没有现成的、通用的"建 handler + 带会话"辅助——它的每个认证相关
// 测试都是就地手写 auth.Session{...} 加 stubAuthenticationService（参见
// TestAuthenticationProtectsAPIAndChecksCSRF、TestDaeInstallRequiresAuthentication）。
// 这里照同一种写法拼一个局部辅助：注入 Authentication 依赖是必须的，
// NewWithDependencies 只有在它非空时才会挂上 authenticationMiddleware
// （见 app.go），不挂上就测不出这个写接口是否被会话校验、CSRF、同源检查这条
// 完整链路保护住。
func newAuthenticatedTestHandler(t *testing.T, dependencies Dependencies) (http.Handler, auth.Session) {
	t.Helper()
	session := auth.Session{
		Token:     "session-token",
		CSRFToken: "csrf-token",
		ExpiresAt: time.Now().Add(time.Hour),
		User:      auth.User{ID: 1, Username: "admin"},
	}
	dependencies.Authentication = &stubAuthenticationService{initialized: true, session: session}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		dependencies,
	)
	if err != nil {
		t.Fatal(err)
	}
	return application.Handler(), session
}

// doJSON 带上会话 Cookie 与 CSRF 头发一个 JSON 请求，与
// TestAuthenticationProtectsAPIAndChecksCSRF 里手工拼请求的写法一致。
func doJSON(t *testing.T, handler http.Handler, session auth.Session, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.Token})
	request.Header.Set("X-CSRF-Token", session.CSRFToken)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestSetGroupPolicyEndpointPassesThrough(t *testing.T) {
	service := &groupPolicyService{}
	handler, session := newAuthenticatedTestHandler(t, Dependencies{Dae: stubDaeService{}, Configuration: service})

	response := doJSON(t, handler, session, http.MethodPut, "/api/v1/groups/proxy/policy",
		`{"policy":"fixed(2)","expectedHash":"旧哈希"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d：%s", response.Code, response.Body.String())
	}
	if service.gotGroup != "proxy" || service.gotPolicy != "fixed(2)" || service.gotHash != "旧哈希" {
		t.Fatalf("参数传递不对: %+v", service)
	}
	var result configstore.SaveResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if result.Hash != "新哈希" {
		t.Fatalf("响应内容不对: %+v", result)
	}
}

func TestSetGroupPolicyEndpointRejectsNonFixed(t *testing.T) {
	service := &groupPolicyService{}
	handler, session := newAuthenticatedTestHandler(t, Dependencies{Dae: stubDaeService{}, Configuration: service})

	for _, policy := range []string{"random", "min_moving_avg", "fixed(-1)", "fixed()", "fixed(1", "", "fixed(1);drop"} {
		response := doJSON(t, handler, session, http.MethodPut, "/api/v1/groups/proxy/policy",
			`{"policy":"`+policy+`","expectedHash":"旧哈希"}`)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("policy=%q 应被拒绝，实际 %d", policy, response.Code)
		}
		if !strings.Contains(response.Body.String(), "group_policy_invalid") {
			t.Fatalf("policy=%q 错误码不对: %s", policy, response.Body.String())
		}
	}
	if service.gotPolicy != "" {
		t.Fatalf("非法请求不该抵达服务层")
	}
}

func TestSetGroupPolicyEndpointRequiresExpectedHash(t *testing.T) {
	service := &groupPolicyService{}
	handler, session := newAuthenticatedTestHandler(t, Dependencies{Dae: stubDaeService{}, Configuration: service})

	response := doJSON(t, handler, session, http.MethodPut, "/api/v1/groups/proxy/policy", `{"policy":"fixed(0)"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("缺少 expectedHash 应被拒绝，实际 %d：%s", response.Code, response.Body.String())
	}
}

func TestSetGroupPolicyEndpointMapsUnlocatable(t *testing.T) {
	service := &groupPolicyService{err: daeconfig.ErrGroupPolicyUnlocatable}
	handler, session := newAuthenticatedTestHandler(t, Dependencies{Dae: stubDaeService{}, Configuration: service})

	response := doJSON(t, handler, session, http.MethodPut, "/api/v1/groups/proxy/policy",
		`{"policy":"fixed(0)","expectedHash":"旧哈希"}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("不可定位应返回 422，实际 %d：%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "group_policy_unlocatable") {
		t.Fatalf("错误码不对: %s", response.Body.String())
	}
}

func TestSetGroupPolicyEndpointMapsConflict(t *testing.T) {
	service := &groupPolicyService{err: configstore.ErrConflict}
	handler, session := newAuthenticatedTestHandler(t, Dependencies{Dae: stubDaeService{}, Configuration: service})

	response := doJSON(t, handler, session, http.MethodPut, "/api/v1/groups/proxy/policy",
		`{"policy":"fixed(0)","expectedHash":"旧哈希"}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("哈希冲突应返回 409，实际 %d", response.Code)
	}
}

func TestSetGroupPolicyEndpointUnavailableWithoutService(t *testing.T) {
	handler, session := newAuthenticatedTestHandler(t, Dependencies{Dae: stubDaeService{}})
	response := doJSON(t, handler, session, http.MethodPut, "/api/v1/groups/proxy/policy",
		`{"policy":"fixed(0)","expectedHash":"旧哈希"}`)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("未初始化配置服务应返回 503，实际 %d", response.Code)
	}
}
