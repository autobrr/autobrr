package domain

import "context"

type UserRepo interface {
	GetUserCount(ctx context.Context) (int, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Store(ctx context.Context, user User) error
	Update(ctx context.Context, user User) error
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	LogDir   string `json:"log_dir"`
}
