package upstream

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOfficialResolveReusesReleaseFromList(t *testing.T) {
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	platform := Platform{Name: "x86_64_v3_avx2", Fallbacks: []string{"x86_64"}}
	asset := AssetName("x86_64")
	apiRequests := 0
	client := testHTTPClient(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch {
		case request.URL.Hostname() == "api.github.com" && strings.HasSuffix(request.URL.Path, "/releases"):
			apiRequests++
			body := fmt.Sprintf(`[{"tag_name":"v2.0.0","published_at":"2026-07-01T00:00:00Z","assets":[{"name":%q,"size":123},{"name":%q,"size":100}]}]`, asset, asset+".dgst")
			return jsonResponse(http.StatusOK, body, nil), nil
		case request.URL.Hostname() == "api.github.com":
			apiRequests++
			return nil, fmt.Errorf("不应重复查询 %s", request.URL)
		case request.URL.Hostname() == "github.com" && strings.HasSuffix(request.URL.Path, ".dgst"):
			return jsonResponse(http.StatusOK, strings.Repeat("a", 64)+"  "+asset+"  sha256\n", nil), nil
		default:
			return nil, fmt.Errorf("意外请求 %s", request.URL)
		}
	}), func() time.Time { return now })
	provider := NewOfficialProvider(client, "daeuniverse", "dae")
	provider.now = func() time.Time { return now }

	versions, err := provider.List(context.Background(), 30)
	if err != nil || len(versions) != 1 {
		t.Fatalf("List = %+v, %v", versions, err)
	}
	resolved, err := provider.Resolve(context.Background(), versions[0].Ref, platform)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Filename != asset || resolved.Platform != "x86_64" || apiRequests != 1 {
		t.Fatalf("Resolve = %+v，API 请求数 = %d", resolved, apiRequests)
	}
}

func TestKdaeResolveReusesRunVerificationFromList(t *testing.T) {
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	platform := Platform{Name: "x86_64_v3_avx2", Fallbacks: []string{"x86_64"}}
	asset := AssetName("x86_64")
	verifyRequests := 0
	client := testHTTPClient(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		path := request.URL.Path
		switch {
		case strings.HasSuffix(path, "/actions/workflows/build.yml/runs"):
			body := `{"workflow_runs":[{"id":123,"head_sha":"abcdef012345","head_branch":"kdae","event":"push","path":".github/workflows/build.yml","created_at":"2026-07-30T00:00:00Z","conclusion":"success","head_commit":{"message":"build"},"head_repository":{"full_name":"olicesx/dae"}}]}`
			return jsonResponse(http.StatusOK, body, nil), nil
		case strings.HasSuffix(path, "/actions/runs/123"):
			verifyRequests++
			return nil, fmt.Errorf("不应重复核验 run")
		case strings.HasSuffix(path, "/actions/runs/123/artifacts"):
			body := fmt.Sprintf(`{"artifacts":[{"name":%q,"size_in_bytes":456,"digest":"sha256:%s","expired":false,"expires_at":"2026-10-30T00:00:00Z"}]}`, asset, strings.Repeat("b", 64))
			return jsonResponse(http.StatusOK, body, nil), nil
		default:
			return nil, fmt.Errorf("意外请求 %s", request.URL)
		}
	}), func() time.Time { return now })
	provider := NewKdaeProvider(client, "olicesx", "dae", "kdae", "build.yml")
	provider.now = func() time.Time { return now }

	versions, err := provider.List(context.Background(), 30)
	if err != nil || len(versions) != 1 {
		t.Fatalf("List = %+v, %v", versions, err)
	}
	resolved, err := provider.Resolve(context.Background(), versions[0].Ref, platform)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Filename != asset || resolved.Platform != "x86_64" || verifyRequests != 0 {
		t.Fatalf("Resolve = %+v，重复核验请求数 = %d", resolved, verifyRequests)
	}
}
