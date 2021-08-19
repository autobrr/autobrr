package domain

type UserRepo interface {
	FindByUsername(username string) (*User, error)
	Store(user User) error
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}
