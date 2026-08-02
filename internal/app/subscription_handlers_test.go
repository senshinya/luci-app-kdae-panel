package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tuoro/kdae-panel/internal/subscriptioncache"
)

type stubSubscriptionNodes struct {
	sources []subscriptioncache.Source
	err     error
}

func (s stubSubscriptionNodes) List(context.Context) ([]subscriptioncache.Source, error) {
	return s.sources, s.err
}

func TestSubscriptionNodeCacheEndpoint(t *testing.T) {
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae: stubDaeService{},
		SubscriptionNodes: stubSubscriptionNodes{sources: []subscriptioncache.Source{{
			Tag: "main", Nodes: []subscriptioncache.Node{{Name: "香港 01", Protocol: "vless", Matches: 1}},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/nodes", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "香港 01") {
		t.Fatalf("订阅节点响应异常: status=%d body=%s", recorder.Code, recorder.Body)
	}
}

func TestSubscriptionNodeCacheEndpointReportsReadFailure(t *testing.T) {
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae: stubDaeService{}, SubscriptionNodes: stubSubscriptionNodes{err: errors.New("permission denied")},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/nodes", nil))
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), "subscription_cache_unavailable") {
		t.Fatalf("订阅节点错误响应异常: status=%d body=%s", recorder.Code, recorder.Body)
	}
}
