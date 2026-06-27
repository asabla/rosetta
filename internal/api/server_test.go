package api

import (
	"bytes"
	"github.com/asabla/rosetta/internal/authz"
	"github.com/asabla/rosetta/internal/openshell"
	"github.com/asabla/rosetta/internal/store"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDryRunCreateDoesNotPersist(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := Server{Store: st, Auth: authz.NewCedarAuthorizer(log), Shell: openshell.LoggingAdapter{Log: log}, Dir: dir}
	req := httptest.NewRequest(http.MethodPost, "/sandboxes", bytes.NewBufferString(`{"principal":"agent","dry_run":true}`))
	rr := httptest.NewRecorder()
	s.Routes().ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("code %d body %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"dry_run":true`) {
		t.Fatal(rr.Body.String())
	}
}
