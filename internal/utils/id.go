package utils

import "github.com/google/uuid"

// GenerateID returns a new random UUID string, used as the canonical
// ID generator across models (ISPs, packages, reviews, etc).
func GenerateID() string {
	return uuid.NewString()
}