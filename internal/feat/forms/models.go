package forms

import (
	"time"

	"github.com/google/uuid"
)

// FormSubmission represents a submitted form entry.
type FormSubmission struct {
	ID        uuid.UUID  `json:"id"`
	SiteID    uuid.UUID  `json:"site_id"`
	FormType  string     `json:"form_type"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Message   string     `json:"message"`
	IPAddress string     `json:"ip_address"`
	UserAgent string     `json:"user_agent"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsRead returns true if the submission has been marked as read.
func (f *FormSubmission) IsRead() bool {
	return f.ReadAt != nil
}

// MessagePreview returns a truncated preview of the message.
func (f *FormSubmission) MessagePreview() string {
	if len(f.Message) <= 80 {
		return f.Message
	}
	return f.Message[:80] + "..."
}
