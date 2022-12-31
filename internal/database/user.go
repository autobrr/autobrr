package database

import (
	"context"
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"
	sq "github.com/Masterminds/squirrel"

	"github.com/rs/zerolog"
)

type UserRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewUserRepo(log logger.Logger, db *DB) domain.UserRepo {
	return &UserRepo{
		log: log.With().Str("repo", "user").Logger(),
		db:  db,
	}
}

func (r *UserRepo) GetUserCount(ctx context.Context) (int, error) {
	queryBuilder := r.db.squirrel.RunWith(r.db.handler).Select("count(*)").From("users")

	row, err := queryBuilder.Query()
	if err != nil {
		return 0, errors.Wrap(err, "error executing query")
	}

	result := 0
	if err := row.Scan(&result); err != nil {
		return 0, errors.Wrap(err, "error scanning row")
	}

	return result, nil
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Select("id", "username", "password").
		From("users").
		Where(sq.Eq{"username": username})

	row, err := queryBuilder.Query()
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var user domain.User

	if err := row.Scan(&user.ID, &user.Username, &user.Password); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	return &user, nil
}

func (r *UserRepo) Store(ctx context.Context, user domain.User) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Insert("users").
		Columns("username", "password").
		Values(user.Username, user.Password)

	if _, err := queryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *UserRepo) Update(ctx context.Context, user domain.User) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Update("users").
		Set("username", user.Username).
		Set("password", user.Password).
		Where(sq.Eq{"username": user.Username})

	if _, err := queryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}
