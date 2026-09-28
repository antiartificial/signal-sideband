package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"signal-sideband/pkg/store"
)

func TestReadinessChecksDatabaseWhileHealthIsLiveness(t *testing.T) {
	db, err := store.NewStore(context.Background(), "postgres://unused:unused@127.0.0.1:1/unavailable?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := NewServer(db, nil, nil, nil, nil, nil, nil, "0", "", "", "test", "test")
	for _, tc := range []struct {
		path string
		want int
	}{
		{path: "/health", want: http.StatusOK},
		{path: "/ready", want: http.StatusServiceUnavailable},
	} {
		response := httptest.NewRecorder()
		server.httpServer.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if response.Code != tc.want {
			t.Errorf("GET %s status = %d, want %d", tc.path, response.Code, tc.want)
		}
	}
}

func TestReadinessWithDatabase(t *testing.T) {
	dsn := os.Getenv("SIGNAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SIGNAL_TEST_DATABASE_URL is not set")
	}
	db, err := store.NewStore(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := NewServer(db, nil, nil, nil, nil, nil, nil, "0", "", "", "test", "test")
	response := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET /ready status = %d, want %d", response.Code, http.StatusOK)
	}
}
