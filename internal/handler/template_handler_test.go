package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// buildTemplateRouter creates a chi router with template handler routes.
// Uses nil service — safe for validation paths that return before calling the service.
func buildTemplateRouter() *chi.Mux {
	r := chi.NewRouter()
	h := NewTemplateHandler(nil)

	r.Post("/templates", h.CreateTemplate)
	r.Get("/templates", h.ListTemplates)
	r.Get("/templates/{id}", h.GetTemplate)
	r.Put("/templates/{id}", h.UpdateTemplate)

	return r
}

func doTemplateReq(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// ─── CreateTemplate ──────────────────────────────────────────────────────────

func TestCreateTemplate_InvalidJSON_Returns400(t *testing.T) {
	r := buildTemplateRouter()
	rr := doTemplateReq(t, r, http.MethodPost, "/templates", `{bad json`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestCreateTemplate_EmptyBody_Returns422(t *testing.T) {
	r := buildTemplateRouter()
	rr := doTemplateReq(t, r, http.MethodPost, "/templates", `{}`)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for empty body (missing required fields), got %d", rr.Code)
	}
}

func TestCreateTemplate_MissingName_Returns422(t *testing.T) {
	r := buildTemplateRouter()
	body := `{"type":"email","subject_template":"Hello","body_template":"<p>Hi</p>"}`
	rr := doTemplateReq(t, r, http.MethodPost, "/templates", body)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for missing name, got %d", rr.Code)
	}
}

func TestCreateTemplate_InvalidType_Returns422(t *testing.T) {
	r := buildTemplateRouter()
	body := `{"name":"welcome","type":"push","subject_template":"Hi","body_template":"<p>Hi</p>"}`
	rr := doTemplateReq(t, r, http.MethodPost, "/templates", body)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid type, got %d", rr.Code)
	}
}

func TestCreateTemplate_MissingBodyTemplate_Returns422(t *testing.T) {
	r := buildTemplateRouter()
	body := `{"name":"welcome","type":"email","subject_template":"Hello"}`
	rr := doTemplateReq(t, r, http.MethodPost, "/templates", body)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for missing body_template, got %d", rr.Code)
	}
}

// ─── GetTemplate ─────────────────────────────────────────────────────────────

func TestGetTemplate_InvalidUUID_Returns400(t *testing.T) {
	r := buildTemplateRouter()
	rr := doTemplateReq(t, r, http.MethodGet, "/templates/not-a-uuid", "")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

// ─── UpdateTemplate ──────────────────────────────────────────────────────────

func TestUpdateTemplate_InvalidUUID_Returns400(t *testing.T) {
	r := buildTemplateRouter()
	rr := doTemplateReq(t, r, http.MethodPut, "/templates/bad-id", `{"name":"updated"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestUpdateTemplate_InvalidJSON_Returns400(t *testing.T) {
	r := buildTemplateRouter()
	rr := doTemplateReq(t, r, http.MethodPut, "/templates/550e8400-e29b-41d4-a716-446655440000", `{bad`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestUpdateTemplate_InvalidType_Returns422(t *testing.T) {
	r := buildTemplateRouter()
	body := `{"type":"fax"}`
	rr := doTemplateReq(t, r, http.MethodPut, "/templates/550e8400-e29b-41d4-a716-446655440000", body)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid type, got %d", rr.Code)
	}
}
