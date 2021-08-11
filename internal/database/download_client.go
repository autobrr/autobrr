package database

import (
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog/log"
)

type DownloadClientRepo struct {
	db *sql.DB
}

func NewDownloadClientRepo(db *sql.DB) domain.DownloadClientRepo {
	return &DownloadClientRepo{db: db}
}

func (r *DownloadClientRepo) List() ([]domain.DownloadClient, error) {

	rows, err := r.db.Query("SELECT id, name, type, enabled, host, port, ssl, username, password FROM client")
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	clients := make([]domain.DownloadClient, 0)

	for rows.Next() {
		var f domain.DownloadClient

		if err := rows.Scan(&f.ID, &f.Name, &f.Type, &f.Enabled, &f.Host, &f.Port, &f.SSL, &f.Username, &f.Password); err != nil {
			log.Error().Err(err)
		}
		if err != nil {
			return nil, err
		}

		clients = append(clients, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return clients, nil
}

func (r *DownloadClientRepo) FindByID(id int32) (*domain.DownloadClient, error) {

	query := `
		SELECT id, name, type, enabled, host, port, ssl, username, password FROM client WHERE id = ?
	`

	row := r.db.QueryRow(query, id)
	if err := row.Err(); err != nil {
		return nil, err
	}

	var client domain.DownloadClient

	if err := row.Scan(&client.ID, &client.Name, &client.Type, &client.Enabled, &client.Host, &client.Port, &client.SSL, &client.Username, &client.Password); err != nil {
		log.Error().Err(err).Msg("could not scan download client to struct")
		return nil, err
	}

	return &client, nil
}

func (r *DownloadClientRepo) FindByActionID(actionID int) ([]domain.DownloadClient, error) {

	rows, err := r.db.Query("SELECT id, name, type, enabled, host, port, ssl, username, password FROM client, action_client WHERE client.id = action_client.client_id AND action_client.action_id = ?", actionID)
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	var clients []domain.DownloadClient
	for rows.Next() {
		var f domain.DownloadClient

		if err := rows.Scan(&f.ID, &f.Name, &f.Type, &f.Enabled, &f.Host, &f.Port, &f.SSL, &f.Username, &f.Password); err != nil {
			log.Error().Err(err)
		}
		if err != nil {
			return nil, err
		}

		clients = append(clients, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return clients, nil
}

func (r *DownloadClientRepo) Store(client domain.DownloadClient) (*domain.DownloadClient, error) {

	var err error
	if client.ID != 0 {
		log.Info().Msg("UPDATE existing record")
		_, err = r.db.Exec(`UPDATE client SET name = ?, type = ?, enabled = ?, host = ?, port = ?, ssl = ?, username = ?, password = ? WHERE id = ?`, client.Name, client.Type, client.Enabled, client.Host, client.Port, client.SSL, client.Username, client.Password, client.ID)
	} else {
		var res sql.Result

		res, err = r.db.Exec(`INSERT INTO client(name, type, enabled, host, port, ssl, username, password)
			VALUES (?, ?, ?, ?, ?, ? , ?, ?) ON CONFLICT DO NOTHING`, client.Name, client.Type, client.Enabled, client.Host, client.Port, client.SSL, client.Username, client.Password)
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}

		resId, _ := res.LastInsertId()
		log.Info().Msgf("LAST INSERT ID %v", resId)
		client.ID = int(resId)
	}

	return &client, nil
}

func (r *DownloadClientRepo) Delete(clientID int) error {
	res, err := r.db.Exec(`DELETE FROM client WHERE client.id = ?`, clientID)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()

	log.Info().Msgf("rows affected %v", rows)

	return nil
}
