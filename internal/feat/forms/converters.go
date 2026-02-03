package forms

import (
	"github.com/cliossg/clio/internal/db/sqlc"
	"github.com/google/uuid"
)

func formSubmissionFromSQLC(f sqlc.FormSubmission) *FormSubmission {
	sub := &FormSubmission{
		ID:        parseUUID(f.ID),
		SiteID:    parseUUID(f.SiteID),
		FormType:  f.FormType,
		Message:   f.Message,
		CreatedAt: f.CreatedAt,
	}
	if f.Name.Valid {
		sub.Name = f.Name.String
	}
	if f.Email.Valid {
		sub.Email = f.Email.String
	}
	if f.IpAddress.Valid {
		sub.IPAddress = f.IpAddress.String
	}
	if f.UserAgent.Valid {
		sub.UserAgent = f.UserAgent.String
	}
	if f.ReadAt.Valid {
		sub.ReadAt = &f.ReadAt.Time
	}
	return sub
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
