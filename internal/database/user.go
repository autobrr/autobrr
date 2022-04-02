package database

import (
	"context"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
)

type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) domain.UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {

	queryBuilder := r.db.squirrel.
		Select("id", "username", "password").
		From("users").
		Where("username = ?", username)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("user.store: error building query")
		return nil, err
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
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

	queryBuilder := r.db.squirrel.
		Update("users").
		Set("username", user.Username).
		Set("password", user.Password).
		Where("username = ?", user.Username)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("user.store: error building query")
		return err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("user.store: error executing query")
		return err
	}

	return err
}
func (r *UserRepo) Update(ctx context.Context, user domain.User) error {

	var err error

	queryBuilder := r.db.squirrel.
		Update("users").
		Set("username", user.Username).
		Set("password", user.Password).
		Where("username = ?", user.Username)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("user.store: error building query")
		return err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("user.store: error executing query")
		return err
	}

	return err
}
