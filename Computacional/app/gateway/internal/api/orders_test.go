package api

import (
	"errors"
	"testing"

	"unbot-gateway/internal/orders"
)

func TestValidationMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		sentinel error
		want     string
	}{
		{
			name:     "extracts safe detail after sentinel prefix",
			err:      errors.New("validation failed: client_user_id is required"),
			sentinel: orders.ErrValidation,
			want:     "client_user_id is required",
		},
		{
			name:     "transition detail preserved",
			err:      errors.New("invalid status transition: cannot transition from pending to delivered"),
			sentinel: orders.ErrInvalidTransition,
			want:     "cannot transition from pending to delivered",
		},
		{
			name:     "no prefix falls back to sentinel text (no internal leak)",
			err:      errors.New("get order: connection refused to 10.0.0.5:5432"),
			sentinel: orders.ErrValidation,
			want:     orders.ErrValidation.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validationMessage(tt.err, tt.sentinel); got != tt.want {
				t.Errorf("validationMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
