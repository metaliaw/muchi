package errorhandler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/search"
)

func TestWriteMapsKnownFailures(t *testing.T) {
	cases := []struct {
		name    string
		failure error
		want    Description
	}{
		{name: "invalid", failure: search.ErrInvalid, want: InvalidRequest()},
		{name: "missing", failure: search.ErrNotFound, want: NotFound()},
		{name: "conflict", failure: search.ErrConflict, want: IdempotencyConflict()},
		{name: "state", failure: search.ErrNotRunning, want: InvalidState()},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			response := httptest.NewRecorder()
			(Handler{}).Write(response, request, "req-1", test.failure)
			if response.Code != test.want.Status || !strings.Contains(response.Body.String(), `"code":"`+test.want.Code+`"`) || !strings.Contains(response.Body.String(), `"request_id":"req-1"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestWriteHidesUnexpectedFailure(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	(Handler{}).Write(response, request, "req-2", errors.New("database password leaked"))
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "password") || !strings.Contains(response.Body.String(), `"incident_id":"inc_`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
