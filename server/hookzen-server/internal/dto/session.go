package dto

import "time"

// SessionResponse masks the raw redis strings into flat, clean device schemas for frontend dashboards.
type SessionResponse struct {
	ID         string    `json:"id"`
	IP         string    `json:"ip"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	DeviceType string    `json:"device_type"`
	City       string    `json:"city"`
	Country    string    `json:"country"`
	IsCurrent  bool      `json:"is_current"` // Extra production touch: Tells the frontend if it's the current device active
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}