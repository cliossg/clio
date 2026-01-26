package model

import (
	"database/sql"
	"time"
)

// NullTime wraps sql.NullTime with convenience methods.
type NullTime struct {
	sql.NullTime
}

// NewNullTime creates a NullTime from a time.Time.
func NewNullTime(t time.Time) NullTime {
	return NullTime{
		NullTime: sql.NullTime{
			Time:  t,
			Valid: true,
		},
	}
}

// NullTimeFromPtr creates a NullTime from a *time.Time.
// Returns an invalid NullTime if the pointer is nil.
func NullTimeFromPtr(t *time.Time) NullTime {
	if t == nil {
		return NullTime{}
	}
	return NewNullTime(*t)
}

// ToPtr converts NullTime to *time.Time.
// Returns nil if the NullTime is not valid.
func (nt NullTime) ToPtr() *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

// Now returns the current time.
func Now() time.Time {
	return time.Now()
}

// NowPtr returns a pointer to the current time.
func NowPtr() *time.Time {
	t := time.Now()
	return &t
}
