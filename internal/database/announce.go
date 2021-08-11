package database

import (
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"
)

type AnnounceRepo struct {
	db *sql.DB
}

func NewAnnounceRepo(db *sql.DB) domain.AnnounceRepo {
	return &AnnounceRepo{db: db}
}

func (a *AnnounceRepo) Store(announce domain.Announce) error {
	return nil
}
