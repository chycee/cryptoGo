package ls

import "crypto_go/internal/event"

// WebhookHandler will handle LS Securities webhook events (Future Implementation)
type WebhookHandler struct {
	inbox chan<- event.Event
}

// NewWebhookHandler creates a placeholder handler
func NewWebhookHandler(inbox chan<- event.Event) *WebhookHandler {
	return &WebhookHandler{inbox: inbox}
}

// TODO: Implement HandleWebhook(w http.ResponseWriter, r *http.Request)
