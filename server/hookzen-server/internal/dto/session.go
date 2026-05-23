package dto

import "time"

type SessionMetadata struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	IP         string    `json:"ip"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	DeviceType string    `json:"device_type"`
	City       string    `json:"city"`
	Country    string    `json:"country"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type SessionResponse struct {
	ID         string    `json:"id"`
	IP         string    `json:"ip"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	DeviceType string    `json:"device_type"`
	City       string    `json:"city"`
	Country    string    `json:"country"`
	IsCurrent  bool      `json:"is_current"`
}

func (s *SessionMetadata) ToResponse(currentSessionID string) *SessionResponse {
	return &SessionResponse{
		ID:         s.ID,
		IP:         s.IP,
		OS:         s.OS,
		Browser:    s.Browser,
		DeviceType: s.DeviceType,
		City:       s.City,
		Country:    s.Country,
		IsCurrent:  s.ID == currentSessionID,
	}
}