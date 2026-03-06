package models

import "time"

type Notification struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	ReadAt    *time.Time `json:"read_at"`
	SentAt    *time.Time `json:"sent_at"`
	CreatedAt time.Time `json:"created_at"`
}
