package model

type Role string

const (
	RoleAdmin Role = "admin"
)

type UserRole struct {
	UserID string
	Role   Role
}
