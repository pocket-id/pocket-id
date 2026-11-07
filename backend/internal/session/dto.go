package session

import (
	"time"

	userAgentParser "github.com/mileusna/useragent"
)

type sessionDto struct {
	ID                   string    `json:"id"`
	Browser              string    `json:"browser"`
	OS                   string    `json:"os"`
	DeviceType           string    `json:"deviceType"`
	UserAgent            string    `json:"userAgent"`
	IPAddress            string    `json:"ipAddress"`
	Country              string    `json:"country"`
	City                 string    `json:"city"`
	AuthenticationMethod string    `json:"authenticationMethod"`
	Current              bool      `json:"current"`
	CreatedAt            time.Time `json:"createdAt"`
	LastActiveAt         time.Time `json:"lastActiveAt"`
	ExpiresAt            time.Time `json:"expiresAt"`
}

func toSessionDtos(sessions []Session, currentSessionID string) []sessionDto {
	dtos := make([]sessionDto, len(sessions))
	for i, session := range sessions {
		ua := userAgentParser.Parse(session.UserAgent)
		deviceType := "desktop"
		switch {
		case ua.Tablet:
			deviceType = "tablet"
		case ua.Mobile:
			deviceType = "mobile"
		}

		dtos[i] = sessionDto{
			ID:                   session.ID,
			Browser:              ua.Name,
			OS:                   ua.OS,
			DeviceType:           deviceType,
			UserAgent:            session.UserAgent,
			IPAddress:            session.IPAddress,
			Country:              session.Country,
			City:                 session.City,
			AuthenticationMethod: session.AuthenticationMethod,
			Current:              session.ID == currentSessionID,
			CreatedAt:            session.CreatedAt,
			LastActiveAt:         session.LastActiveAt,
			ExpiresAt:            session.endsAt(),
		}
	}
	return dtos
}
