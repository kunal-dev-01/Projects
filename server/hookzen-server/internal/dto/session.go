package dto

import "time"


type SessionResponse struct {
	ID         string    `json:"id"`
	IP         string    `json:"ip"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	DeviceType string    `json:"device_type"`
	City       string    `json:"city"`
	Country    string    `json:"country"`
	IsCurrent  bool      `json:"is_current"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}