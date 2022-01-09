package database

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) domain.UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, username, password FROM users WHERE username = ?`

	row := r.db.QueryRowContext(ctx, query, username)
	if err := row.Err(); err != nil {
		return nil, err
	}

	var user domain.User

	if err := row.Scan(&user.ID, &user.Username, &user.Password); err != nil {
		log.Error().Err(err).Msg("could not scan user to struct")
		return nil, err
	}

	return &user, nil
}

func (r *UserRepo) Store(ctx context.Context, user domain.User) error {

	var err error
	if user.ID != 0 {
		update := `UPDATE users SET password = ? WHERE username = ?`
		_, err = r.db.ExecContext(ctx, update, user.Password, user.Username)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error executing query")
			return err
		}

	} else {
		query := `INSERT INTO users (username, password) VALUES (?, ?)`
		_, err = r.db.ExecContext(ctx, query, user.Username, user.Password)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error executing query")
			return err
		}
	}

	return err
}
