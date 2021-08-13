package domain

type UserRepo interface {
	FindByUsername(username string) (*User, error)
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
