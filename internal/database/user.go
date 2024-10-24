// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

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
	queryBuilder := r.db.squirrel.Select("count(*)").From("users")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
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
		Select("id", "username", "password", "two_factor_auth", "tfa_secret").
		From("users").
		Where(sq.Eq{"username": username})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var user domain.User

	if err := row.Scan(&user.ID, &user.Username, &user.Password, &user.TwoFactorAuth, &user.TFASecret); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	return &user, nil
}

func (r *UserRepo) Store(ctx context.Context, req domain.CreateUserRequest) error {
	queryBuilder := r.db.squirrel.
		Insert("users").
		Columns("username", "password", "two_factor_auth", "tfa_secret").
		Values(req.Username, req.Password, false, "")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return err
}

func (r *UserRepo) Update(ctx context.Context, user domain.UpdateUserRequest) error {
	queryBuilder := r.db.squirrel.Update("users")

	if user.UsernameNew != "" {
		queryBuilder = queryBuilder.Set("username", user.UsernameNew)
	}

	if user.PasswordNewHash != "" {
		queryBuilder = queryBuilder.Set("password", user.PasswordNewHash)
	}

	queryBuilder = queryBuilder.Where(sq.Eq{"username": user.UsernameCurrent})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *UserRepo) Delete(ctx context.Context, username string) error {
	queryBuilder := r.db.squirrel.
		Delete("users").
		Where(sq.Eq{"username": username})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("user.delete: successfully deleted user: %s", username)

	return nil
}

func (r *UserRepo) Store2FASecret(ctx context.Context, username string, secret string) error {
	// Store the provided secret but don't enable 2FA yet
	queryBuilder := r.db.squirrel.
		Update("users").
		Set("tfa_secret", secret).
		Where(sq.Eq{"username": username})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *UserRepo) Enable2FA(ctx context.Context, username string, secret string) error {
	queryBuilder := r.db.squirrel.
		Update("users").
		Set("two_factor_auth", true).
		Set("tfa_secret", secret).
		Where(sq.Eq{"username": username})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *UserRepo) Verify2FA(ctx context.Context, username string, code string) error {
	// This is handled at the service layer
	return nil
}

func (r *UserRepo) Get2FASecret(ctx context.Context, username string) (string, error) {
	queryBuilder := r.db.squirrel.
		Select("tfa_secret").
		From("users").
		Where(sq.Eq{"username": username})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return "", errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return "", errors.Wrap(err, "error executing query")
	}

	var secret string
	if err := row.Scan(&secret); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrRecordNotFound
		}
		return "", errors.Wrap(err, "error scanning row")
	}

	return secret, nil
}

func (r *UserRepo) Disable2FA(ctx context.Context, username string) error {
	queryBuilder := r.db.squirrel.
		Update("users").
		Set("two_factor_auth", false).
		Set("tfa_secret", "").
		Where(sq.Eq{"username": username})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}
