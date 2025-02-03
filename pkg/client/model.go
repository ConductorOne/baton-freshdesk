package client

import "time"

type Agent struct {
	ID             int       `json:"id,omitempty"`
	Available      bool      `json:"available,omitempty"`
	AvailableSince time.Time `json:"available_since,omitempty"`
	Occasional     bool      `json:"occasional,omitempty"`
	Signature      string    `json:"signature,omitempty"`
	TicketScope    int       `json:"ticket_scope,omitempty"`
	Type           string    `json:"type,omitempty"`
	SkillIDs       []int     `json:"skill_ids,omitempty"`
	GroupIDs       []int     `json:"group_ids,omitempty"`
	RoleIDs        []int     `json:"role_ids,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	Contact        Contact   `json:"contact,omitempty"`
	FocusMode      bool      `json:"focus_mode,omitempty"`
}

type Contact struct {
	Active      bool      `json:"active,omitempty"`
	Email       string    `json:"email,omitempty"`
	JobTitle    string    `json:"job_title,omitempty"`
	Language    string    `json:"language,omitempty"`
	LastLoginAt time.Time `json:"last_login_at,omitempty"`
	Mobile      string    `json:"mobile,omitempty"`
	Name        string    `json:"name,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	TimeZone    string    `json:"time_zone,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type Role struct {
	ID          int       `json:"id,omitempty"`
	Description string    `json:"description,omitempty"`
	Name        string    `json:"name,omitempty"`
	Default     bool      `json:"default,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type Group struct {
	ID               int       `json:"id,omitempty"`
	AgentIDs         []int64   `json:"agent_ids,omitempty"`
	AutoTicketAssign int       `json:"auto_ticket_assign,omitempty"`
	BusinessHourID   int       `json:"business_hour_id,omitempty"`
	Description      string    `json:"description,omitempty"`
	EscalateTo       int       `json:"escalate_to,omitempty"`
	Name             string    `json:"name,omitempty"`
	UnassignedFor    string    `json:"unassigned_for,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}
