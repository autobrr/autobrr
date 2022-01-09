package domain

import "context"

type UserRepo interface {
	FindByUsername(ctx context.Context, username string) (*User, error)
	Store(ctx context.Context, user User) error
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}
