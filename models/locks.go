package models

type LockOrUnlockRequest struct {
	Paths []string `json:"paths" validate:"required"`
}
