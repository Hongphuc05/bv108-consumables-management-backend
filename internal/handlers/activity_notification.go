package handlers

import (
	"fmt"
	"strings"
	"time"

	"bv108-consumables-management-backend/internal/realtime"
)

type ActivityNotificationPayload struct {
	ID         string `json:"id"`
	Category   string `json:"category,omitempty"`
	Action     string `json:"action"`
	ActorID    int64  `json:"actorId,omitempty"`
	ActorName  string `json:"actorName,omitempty"`
	ActorEmail string `json:"actorEmail,omitempty"`
	Count      int    `json:"count,omitempty"`
	Status     string `json:"status,omitempty"`
	Month      int    `json:"month,omitempty"`
	Year       int    `json:"year,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

func broadcastActivityNotification(hub *realtime.Hub, payload ActivityNotificationPayload) {
	if hub == nil {
		return
	}

	payload.Action = strings.TrimSpace(payload.Action)
	if payload.Action == "" {
		return
	}

	payload.ActorName = strings.TrimSpace(payload.ActorName)
	payload.ActorEmail = strings.TrimSpace(payload.ActorEmail)
	payload.Category = strings.TrimSpace(payload.Category)
	payload.Status = strings.TrimSpace(payload.Status)

	createdAt := strings.TrimSpace(payload.CreatedAt)
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	payload.CreatedAt = createdAt

	if strings.TrimSpace(payload.ID) == "" {
		payload.ID = fmt.Sprintf("%s:%d:%d:%d", payload.Action, payload.ActorID, payload.Count, time.Now().UTC().UnixNano())
	}

	hub.Broadcast("notifications.activity", payload)
}
