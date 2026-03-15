package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/institutoitinerante/notification-service/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resendStubServer returns a test HTTP server that responds with the given
// status and body to every POST — used to simulate the Resend API.
func resendStubServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// newSvcWithServer creates a NotificationService whose httpClient is wired to
// the test server via a RoundTripper that rewrites every request host.
func newSvcWithServer(
	notifRepo *mocks.MockNotificationRepository,
	tmplRepo *mocks.MockTemplateRepository,
	srv *httptest.Server,
) *NotificationService {
	svc := NewNotificationService(notifRepo, tmplRepo, "re_test_key", "noreply@example.com")
	svc.httpClient = &http.Client{
		Transport: &hostOverrideTransport{target: srv.URL, inner: srv.Client().Transport},
	}
	return svc
}

// hostOverrideTransport rewrites every outgoing request to point at `target`
// (the httptest.Server URL) regardless of the original host.
type hostOverrideTransport struct {
	target string
	inner  http.RoundTripper
}

func (t *hostOverrideTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Build a new request with the same method/headers/body but pointing at
	// our local test server.
	clone := req.Clone(req.Context())
	// target is like "http://127.0.0.1:PORT"
	host := strings.TrimPrefix(t.target, "http://")
	host = strings.TrimPrefix(host, "https://")
	clone.URL.Scheme = "http"
	clone.URL.Host = host
	return t.inner.RoundTrip(clone)
}

// ─── renderTemplate ───────────────────────────────────────────────────────────

func TestRenderTemplate_ReplacesPlaceholders(t *testing.T) {
	tmpl := "Hello {{name}}, your code is {{code}}."
	vars := map[string]interface{}{"name": "Alice", "code": "999"}

	result := renderTemplate(tmpl, vars)

	assert.Equal(t, "Hello Alice, your code is 999.", result)
}

func TestRenderTemplate_NoPlaceholders(t *testing.T) {
	result := renderTemplate("Static content", nil)
	assert.Equal(t, "Static content", result)
}

func TestRenderTemplate_MissingVariable_LeavesPlaceholder(t *testing.T) {
	result := renderTemplate("Hello {{name}}, welcome to {{app}}.", map[string]interface{}{"name": "Bob"})
	assert.Contains(t, result, "Bob")
	assert.Contains(t, result, "{{app}}", "unreplaced placeholder must remain")
}

func TestRenderTemplate_MultipleOccurrences(t *testing.T) {
	result := renderTemplate("{{x}} and {{x}}", map[string]interface{}{"x": "go"})
	assert.Equal(t, "go and go", result)
}

// ─── Send: no template ────────────────────────────────────────────────────────

func TestNotificationSend_EmailSuccess(t *testing.T) {
	srv := resendStubServer(http.StatusOK, `{"id":"msg_1"}`)
	defer srv.Close()

	notifID := uuid.New()
	notifRepo := &mocks.MockNotificationRepository{
		CreateFn: func(_ context.Context, n *model.Notification) (*model.Notification, error) {
			n.ID = notifID
			return n, nil
		},
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Notification, error) {
			return &model.Notification{ID: id, Status: model.NotificationStatusSent}, nil
		},
	}
	svc := newSvcWithServer(notifRepo, &mocks.MockTemplateRepository{}, srv)

	notification, err := svc.Send(context.Background(), &model.SendNotificationRequest{
		Type:      model.NotificationTypeEmail,
		Recipient: "user@example.com",
		Subject:   "Hello",
		Body:      "<p>Test</p>",
	})

	require.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, model.NotificationStatusSent, notification.Status)
}

func TestNotificationSend_CreateError_ReturnsError(t *testing.T) {
	notifRepo := &mocks.MockNotificationRepository{
		CreateFn: func(_ context.Context, _ *model.Notification) (*model.Notification, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewNotificationService(notifRepo, &mocks.MockTemplateRepository{}, "re_key", "from@example.com")

	_, err := svc.Send(context.Background(), &model.SendNotificationRequest{
		Type:      model.NotificationTypeEmail,
		Recipient: "user@example.com",
		Subject:   "Test",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create notification")
}

// ─── Send: with template ─────────────────────────────────────────────────────

func TestNotificationSend_WithTemplate_RendersSubjectAndBody(t *testing.T) {
	srv := resendStubServer(http.StatusOK, `{"id":"msg_2"}`)
	defer srv.Close()

	tmplID := uuid.New()
	notifRepo := &mocks.MockNotificationRepository{
		CreateFn: func(_ context.Context, n *model.Notification) (*model.Notification, error) {
			return n, nil
		},
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Notification, error) {
			return &model.Notification{ID: id, Status: model.NotificationStatusSent}, nil
		},
	}
	tmplRepo := &mocks.MockTemplateRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Template, error) {
			return &model.Template{
				ID:              id,
				SubjectTemplate: "Welcome {{name}}",
				BodyTemplate:    "<p>Hello {{name}}</p>",
			}, nil
		},
	}
	svc := newSvcWithServer(notifRepo, tmplRepo, srv)

	notification, err := svc.Send(context.Background(), &model.SendNotificationRequest{
		Type:       model.NotificationTypeEmail,
		Recipient:  "user@example.com",
		TemplateID: &tmplID,
		Variables:  map[string]interface{}{"name": "Alice"},
	})

	require.NoError(t, err)
	assert.NotNil(t, notification)
}

func TestNotificationSend_TemplateGetError_ReturnsError(t *testing.T) {
	tmplID := uuid.New()
	tmplRepo := &mocks.MockTemplateRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Template, error) {
			return nil, errors.New("not found")
		},
	}
	svc := NewNotificationService(&mocks.MockNotificationRepository{}, tmplRepo, "key", "from@example.com")

	_, err := svc.Send(context.Background(), &model.SendNotificationRequest{
		Type:       model.NotificationTypeEmail,
		Recipient:  "user@example.com",
		TemplateID: &tmplID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get template")
}

func TestNotificationSend_TemplateNilResult_ReturnsError(t *testing.T) {
	tmplID := uuid.New()
	tmplRepo := &mocks.MockTemplateRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Template, error) {
			return nil, nil // repo returned nil without error
		},
	}
	svc := NewNotificationService(&mocks.MockNotificationRepository{}, tmplRepo, "key", "from@example.com")

	_, err := svc.Send(context.Background(), &model.SendNotificationRequest{
		Type:       model.NotificationTypeEmail,
		Recipient:  "user@example.com",
		TemplateID: &tmplID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

// ─── Send: SMS (not implemented) ─────────────────────────────────────────────

func TestNotificationSend_SMSType_MarkedFailed(t *testing.T) {
	notifID := uuid.New()
	notifRepo := &mocks.MockNotificationRepository{
		CreateFn: func(_ context.Context, n *model.Notification) (*model.Notification, error) {
			n.ID = notifID
			return n, nil
		},
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Notification, error) {
			return &model.Notification{ID: id, Status: model.NotificationStatusFailed}, nil
		},
	}
	svc := NewNotificationService(notifRepo, &mocks.MockTemplateRepository{}, "re_key", "from@example.com")

	notification, err := svc.Send(context.Background(), &model.SendNotificationRequest{
		Type:      model.NotificationTypeSMS,
		Recipient: "+5511999999999",
		Subject:   "Test",
	})

	require.NoError(t, err)
	assert.Equal(t, model.NotificationStatusFailed, notification.Status)
}

// ─── Send: Resend API errors ──────────────────────────────────────────────────

func TestNotificationSend_ResendReturns4xx_MarkedFailed(t *testing.T) {
	srv := resendStubServer(http.StatusUnauthorized, `{"error":"invalid api key"}`)
	defer srv.Close()

	notifID := uuid.New()
	notifRepo := &mocks.MockNotificationRepository{
		CreateFn: func(_ context.Context, n *model.Notification) (*model.Notification, error) {
			n.ID = notifID
			return n, nil
		},
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Notification, error) {
			return &model.Notification{ID: id, Status: model.NotificationStatusFailed}, nil
		},
	}
	svc := newSvcWithServer(notifRepo, &mocks.MockTemplateRepository{}, srv)

	notification, err := svc.Send(context.Background(), &model.SendNotificationRequest{
		Type:      model.NotificationTypeEmail,
		Recipient: "user@example.com",
		Subject:   "Test",
	})

	require.NoError(t, err) // Send doesn't fail — it records failure in DB
	assert.Equal(t, model.NotificationStatusFailed, notification.Status)
}

// ─── GetNotification ──────────────────────────────────────────────────────────

func TestGetNotification_ReturnsFromRepo(t *testing.T) {
	id := uuid.New()
	notifRepo := &mocks.MockNotificationRepository{
		GetByIDFn: func(_ context.Context, gotID uuid.UUID) (*model.Notification, error) {
			return &model.Notification{ID: gotID, Status: model.NotificationStatusSent}, nil
		},
	}
	svc := NewNotificationService(notifRepo, &mocks.MockTemplateRepository{}, "key", "from@example.com")

	n, err := svc.GetNotification(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, id, n.ID)
}

func TestGetNotification_Error_PropagatesError(t *testing.T) {
	notifRepo := &mocks.MockNotificationRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Notification, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewNotificationService(notifRepo, &mocks.MockTemplateRepository{}, "key", "from@example.com")

	_, err := svc.GetNotification(context.Background(), uuid.New())

	require.Error(t, err)
}

// ─── ListNotifications ────────────────────────────────────────────────────────

func TestListNotifications_ReturnsFromRepo(t *testing.T) {
	notifRepo := &mocks.MockNotificationRepository{
		ListFn: func(_ context.Context, _ *model.NotificationFilter) (*model.NotificationListResponse, error) {
			return &model.NotificationListResponse{
				Notifications: []model.Notification{
					{ID: uuid.New()},
					{ID: uuid.New()},
				},
				Total: 2,
			}, nil
		},
	}
	svc := NewNotificationService(notifRepo, &mocks.MockTemplateRepository{}, "key", "from@example.com")

	result, err := svc.ListNotifications(context.Background(), &model.NotificationFilter{Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Notifications, 2)
}

func TestListNotifications_Error_PropagatesError(t *testing.T) {
	notifRepo := &mocks.MockNotificationRepository{
		ListFn: func(_ context.Context, _ *model.NotificationFilter) (*model.NotificationListResponse, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewNotificationService(notifRepo, &mocks.MockTemplateRepository{}, "key", "from@example.com")

	_, err := svc.ListNotifications(context.Background(), &model.NotificationFilter{})

	require.Error(t, err)
}

// ─── RetryPending ─────────────────────────────────────────────────────────────

func TestRetryPending_NoNotifications_NoError(t *testing.T) {
	notifRepo := &mocks.MockNotificationRepository{
		GetPendingRetriesFn: func(_ context.Context) ([]model.Notification, error) {
			return []model.Notification{}, nil
		},
	}
	svc := NewNotificationService(notifRepo, &mocks.MockTemplateRepository{}, "key", "from@example.com")

	err := svc.RetryPending(context.Background())

	require.NoError(t, err)
}

func TestRetryPending_GetPendingError_ReturnsError(t *testing.T) {
	notifRepo := &mocks.MockNotificationRepository{
		GetPendingRetriesFn: func(_ context.Context) ([]model.Notification, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewNotificationService(notifRepo, &mocks.MockTemplateRepository{}, "key", "from@example.com")

	err := svc.RetryPending(context.Background())

	require.Error(t, err)
}

func TestRetryPending_WithPendingNotifications_AttemptsRetry(t *testing.T) {
	srv := resendStubServer(http.StatusOK, `{"id":"retry_1"}`)
	defer srv.Close()

	notifID := uuid.New()
	updateCalled := false
	notifRepo := &mocks.MockNotificationRepository{
		GetPendingRetriesFn: func(_ context.Context) ([]model.Notification, error) {
			return []model.Notification{
				{
					ID:        notifID,
					Type:      model.NotificationTypeEmail,
					Recipient: "user@example.com",
					Subject:   "Retry Test",
					Body:      "<p>Retry</p>",
					Status:    model.NotificationStatusRetrying,
					Attempts:  1,
					CreatedAt: time.Now(),
				},
			}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ model.NotificationStatus, _ int, _ *string, _ *time.Time) error {
			updateCalled = true
			return nil
		},
	}
	svc := newSvcWithServer(notifRepo, &mocks.MockTemplateRepository{}, srv)

	err := svc.RetryPending(context.Background())

	require.NoError(t, err)
	assert.True(t, updateCalled, "UpdateStatus must be called when retry succeeds")
}
