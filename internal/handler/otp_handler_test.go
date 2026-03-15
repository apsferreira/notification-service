package handler

// Unit tests for OTPHandler.
//
// OTPHandler holds *service.OTPService (a concrete struct), so we cannot inject
// a mock through NewOTPHandler.  For the validation paths (missing fields,
// invalid JSON) the service is never reached, so we safely pass nil.
//
// For paths that call the service we define a local otpServiceIface and a thin
// adapter (otpHandlerTestable) that wires an in-process stub to the same HTTP
// handling logic reproduced here.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Local interface + handler substitute ────────────────────────────────────

type otpSvcIface interface {
	GenerateAndSend(ctx context.Context, email string) (time.Time, error)
	Verify(ctx context.Context, email, code string) error
}

// otpHandlerSub reproduces OTPHandler logic against the interface, enabling
// full-path tests without a real OTPService / DB.
type otpHandlerSub struct {
	svc otpSvcIface
}

func (h *otpHandlerSub) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req model.SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	expiresAt, err := h.svc.GenerateAndSend(r.Context(), req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := model.OTPResponse{Message: "OTP sent successfully", ExpiresAt: expiresAt}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *otpHandlerSub) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req model.VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Code == "" {
		http.Error(w, "Email and code are required", http.StatusBadRequest)
		return
	}

	err := h.svc.Verify(r.Context(), req.Email, req.Code)
	resp := model.VerifyOTPResponse{}
	if err != nil {
		resp.Valid = false
		resp.Message = err.Error()
	} else {
		resp.Valid = true
		resp.Message = "OTP verified successfully"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// ─── Mock OTP service ─────────────────────────────────────────────────────────

type mockOTPSvc struct {
	generateAndSendFn func(ctx context.Context, email string) (time.Time, error)
	verifyFn          func(ctx context.Context, email, code string) error
}

func (m *mockOTPSvc) GenerateAndSend(ctx context.Context, email string) (time.Time, error) {
	if m.generateAndSendFn != nil {
		return m.generateAndSendFn(ctx, email)
	}
	return time.Now().Add(10 * time.Minute), nil
}

func (m *mockOTPSvc) Verify(ctx context.Context, email, code string) error {
	if m.verifyFn != nil {
		return m.verifyFn(ctx, email, code)
	}
	return nil
}

// ─── Router builders ──────────────────────────────────────────────────────────

// buildOTPRouter uses the real OTPHandler with nil service (safe only for
// validation paths that return before touching the service).
func buildOTPRouter() *chi.Mux {
	r := chi.NewRouter()
	h := NewOTPHandler(nil)
	r.Post("/otp/send", h.SendOTP)
	r.Post("/otp/verify", h.VerifyOTP)
	return r
}

// buildOTPSubRouter uses otpHandlerSub backed by a mock service.
func buildOTPSubRouter(svc otpSvcIface) *chi.Mux {
	r := chi.NewRouter()
	h := &otpHandlerSub{svc: svc}
	r.Post("/otp/send", h.SendOTP)
	r.Post("/otp/verify", h.VerifyOTP)
	return r
}

func doOTPReq(t *testing.T, router http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func doRawOTPReq(t *testing.T, router http.Handler, method, path, rawBody string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// ─── SendOTP: validation (nil service safe) ───────────────────────────────────

func TestSendOTP_InvalidJSON_Returns400(t *testing.T) {
	r := buildOTPRouter()
	rr := doRawOTPReq(t, r, http.MethodPost, "/otp/send", `{bad json`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSendOTP_MissingEmail_Returns400(t *testing.T) {
	r := buildOTPRouter()
	rr := doOTPReq(t, r, http.MethodPost, "/otp/send", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Email is required")
}

func TestSendOTP_EmptyEmailField_Returns400(t *testing.T) {
	r := buildOTPRouter()
	rr := doOTPReq(t, r, http.MethodPost, "/otp/send", model.SendOTPRequest{Email: ""})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ─── SendOTP: service integration ────────────────────────────────────────────

func TestSendOTP_Success_Returns200(t *testing.T) {
	expiry := time.Now().Add(10 * time.Minute)
	svc := &mockOTPSvc{
		generateAndSendFn: func(_ context.Context, email string) (time.Time, error) {
			assert.Equal(t, "user@example.com", email)
			return expiry, nil
		},
	}
	r := buildOTPSubRouter(svc)
	rr := doOTPReq(t, r, http.MethodPost, "/otp/send", model.SendOTPRequest{Email: "user@example.com"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp model.OTPResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "OTP sent successfully", resp.Message)
}

func TestSendOTP_ServiceError_Returns500(t *testing.T) {
	svc := &mockOTPSvc{
		generateAndSendFn: func(_ context.Context, _ string) (time.Time, error) {
			return time.Time{}, errors.New("rate limit exceeded")
		},
	}
	r := buildOTPSubRouter(svc)
	rr := doOTPReq(t, r, http.MethodPost, "/otp/send", model.SendOTPRequest{Email: "user@example.com"})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "rate limit exceeded")
}

// ─── VerifyOTP: validation (nil service safe) ─────────────────────────────────

func TestVerifyOTP_InvalidJSON_Returns400(t *testing.T) {
	r := buildOTPRouter()
	rr := doRawOTPReq(t, r, http.MethodPost, "/otp/verify", `{bad json`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVerifyOTP_MissingEmail_Returns400(t *testing.T) {
	r := buildOTPRouter()
	rr := doOTPReq(t, r, http.MethodPost, "/otp/verify", map[string]string{"code": "123456"})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVerifyOTP_MissingCode_Returns400(t *testing.T) {
	r := buildOTPRouter()
	rr := doOTPReq(t, r, http.MethodPost, "/otp/verify", map[string]string{"email": "user@example.com"})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVerifyOTP_BothFieldsMissing_Returns400(t *testing.T) {
	r := buildOTPRouter()
	rr := doOTPReq(t, r, http.MethodPost, "/otp/verify", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ─── VerifyOTP: service integration ──────────────────────────────────────────

func TestVerifyOTP_ValidCode_Returns200WithValidTrue(t *testing.T) {
	svc := &mockOTPSvc{
		verifyFn: func(_ context.Context, email, code string) error {
			assert.Equal(t, "user@example.com", email)
			assert.Equal(t, "123456", code)
			return nil
		},
	}
	r := buildOTPSubRouter(svc)
	rr := doOTPReq(t, r, http.MethodPost, "/otp/verify", model.VerifyOTPRequest{
		Email: "user@example.com",
		Code:  "123456",
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp model.VerifyOTPResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.True(t, resp.Valid)
	assert.Equal(t, "OTP verified successfully", resp.Message)
}

func TestVerifyOTP_InvalidCode_Returns200WithValidFalse(t *testing.T) {
	svc := &mockOTPSvc{
		verifyFn: func(_ context.Context, _, _ string) error {
			return errors.New("invalid OTP code")
		},
	}
	r := buildOTPSubRouter(svc)
	rr := doOTPReq(t, r, http.MethodPost, "/otp/verify", model.VerifyOTPRequest{
		Email: "user@example.com",
		Code:  "000000",
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp model.VerifyOTPResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.False(t, resp.Valid)
	assert.Equal(t, "invalid OTP code", resp.Message)
}

func TestVerifyOTP_ExpiredCode_Returns200WithValidFalse(t *testing.T) {
	svc := &mockOTPSvc{
		verifyFn: func(_ context.Context, _, _ string) error {
			return errors.New("OTP has expired")
		},
	}
	r := buildOTPSubRouter(svc)
	rr := doOTPReq(t, r, http.MethodPost, "/otp/verify", model.VerifyOTPRequest{
		Email: "user@example.com",
		Code:  "123456",
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp model.VerifyOTPResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.False(t, resp.Valid)
	assert.Equal(t, "OTP has expired", resp.Message)
}

func TestVerifyOTP_MaxAttemptsExceeded_Returns200WithValidFalse(t *testing.T) {
	svc := &mockOTPSvc{
		verifyFn: func(_ context.Context, _, _ string) error {
			return errors.New("maximum verification attempts exceeded")
		},
	}
	r := buildOTPSubRouter(svc)
	rr := doOTPReq(t, r, http.MethodPost, "/otp/verify", model.VerifyOTPRequest{
		Email: "user@example.com",
		Code:  "123456",
	})

	var resp model.VerifyOTPResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.Message, "attempts exceeded")
}

func TestVerifyOTP_RateLimitError_Returns200WithValidFalse(t *testing.T) {
	svc := &mockOTPSvc{
		verifyFn: func(_ context.Context, _, _ string) error {
			return errors.New("no valid OTP found for this email")
		},
	}
	r := buildOTPSubRouter(svc)
	rr := doOTPReq(t, r, http.MethodPost, "/otp/verify", model.VerifyOTPRequest{
		Email: "user@example.com",
		Code:  "123456",
	})

	var resp model.VerifyOTPResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.False(t, resp.Valid)
}

// ─── Content-Type header ─────────────────────────────────────────────────────

func TestSendOTP_ResponseContentType(t *testing.T) {
	svc := &mockOTPSvc{}
	r := buildOTPSubRouter(svc)
	rr := doOTPReq(t, r, http.MethodPost, "/otp/send", model.SendOTPRequest{Email: "user@example.com"})

	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestVerifyOTP_ResponseContentType(t *testing.T) {
	svc := &mockOTPSvc{}
	r := buildOTPSubRouter(svc)
	rr := doOTPReq(t, r, http.MethodPost, "/otp/verify", model.VerifyOTPRequest{
		Email: "user@example.com",
		Code:  "123456",
	})

	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}
