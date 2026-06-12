package swagger

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
)

func TestRouterExposesExpectedRoutes(t *testing.T) {
	router := NewRouter()

	testCases := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/"},
		{method: http.MethodGet, path: "/api/sql"},
		{method: http.MethodGet, path: "/api/addresssummary"},
		{method: http.MethodGet, path: "/api/status"},
		{method: http.MethodGet, path: "/api/validate"},
		{method: http.MethodGet, path: "/api/process"},
		{method: http.MethodGet, path: "/api/sync/name"},
		{method: http.MethodGet, path: "/api/sync/addresses"},
		{method: http.MethodGet, path: "/api/sync/txvalues"},
		{method: http.MethodGet, path: "/metrics"},
	}

	for _, testCase := range testCases {
		request := newRouteRequest(t, testCase.method, testCase.path)
		if !router.Match(request, &mux.RouteMatch{}) {
			t.Fatalf("expected %s %s to be registered", testCase.method, testCase.path)
		}
	}
}

func TestRouterDoesNotExposeAutoUpdate(t *testing.T) {
	router := NewRouter()
	request := newRouteRequest(t, http.MethodPost, "/api/autoupdate")
	if router.Match(request, &mux.RouteMatch{}) {
		t.Fatal("expected POST /api/autoupdate to be removed")
	}
}

func newRouteRequest(t *testing.T, method string, path string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return request
}
