package database

import (
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog/log"
)

type ActionRepo struct {
	db *sql.DB
}

func NewActionRepo(db *sql.DB) domain.ActionRepo {
	return &ActionRepo{db: db}
}

func (r *ActionRepo) FindByFilterID(filterID int) ([]domain.Action, error) {

	rows, err := r.db.Query("SELECT id, name, type, enabled, exec_cmd, exec_args, watch_folder, category, tags, label, save_path, paused, ignore_rules, limit_download_speed, limit_upload_speed, client_id FROM action WHERE action.filter_id = ?", filterID)
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	var actions []domain.Action
	for rows.Next() {
		var a domain.Action

		var execCmd, execArgs, watchFolder, category, tags, label, savePath sql.NullString
		var limitUl, limitDl sql.NullInt64
		var clientID sql.NullInt32
		// filterID
		var paused, ignoreRules sql.NullBool

		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Enabled, &execCmd, &execArgs, &watchFolder, &category, &tags, &label, &savePath, &paused, &ignoreRules, &limitDl, &limitUl, &clientID); err != nil {
			log.Fatal().Err(err)
		}
		if err != nil {
			return nil, err
		}

		a.ExecCmd = execCmd.String
		a.ExecArgs = execArgs.String
		a.WatchFolder = watchFolder.String
		a.Category = category.String
		a.Tags = tags.String
		a.Label = label.String
		a.SavePath = savePath.String
		a.Paused = paused.Bool
		a.IgnoreRules = ignoreRules.Bool
		a.LimitUploadSpeed = limitUl.Int64
		a.LimitDownloadSpeed = limitDl.Int64
		a.ClientID = clientID.Int32

		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return actions, nil
}

func (r *ActionRepo) List() ([]domain.Action, error) {

	rows, err := r.db.Query("SELECT id, name, type, enabled, exec_cmd, exec_args, watch_folder, category, tags, label, save_path, paused, ignore_rules, limit_download_speed, limit_upload_speed, client_id FROM action")
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	var actions []domain.Action
	for rows.Next() {
		var a domain.Action

		var execCmd, execArgs, watchFolder, category, tags, label, savePath sql.NullString
		var limitUl, limitDl sql.NullInt64
		var clientID sql.NullInt32
		var paused, ignoreRules sql.NullBool

		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Enabled, &execCmd, &execArgs, &watchFolder, &category, &tags, &label, &savePath, &paused, &ignoreRules, &limitDl, &limitUl, &clientID); err != nil {
			log.Fatal().Err(err)
		}
		if err != nil {
			return nil, err
		}

		a.Category = category.String
		a.Tags = tags.String
		a.Label = label.String
		a.SavePath = savePath.String
		a.Paused = paused.Bool
		a.IgnoreRules = ignoreRules.Bool
		a.LimitUploadSpeed = limitUl.Int64
		a.LimitDownloadSpeed = limitDl.Int64
		a.ClientID = clientID.Int32

		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return actions, nil
}

func (r *ActionRepo) Delete(actionID int) error {
	res, err := r.db.Exec(`DELETE FROM action WHERE action.id = ?`, actionID)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()

	log.Info().Msgf("rows affected %v", rows)

	return nil
}

func (r *ActionRepo) Store(action domain.Action) (*domain.Action, error) {

	execCmd := toNullString(action.ExecCmd)
	execArgs := toNullString(action.ExecArgs)
	watchFolder := toNullString(action.WatchFolder)
	category := toNullString(action.Category)
	tags := toNullString(action.Tags)
	label := toNullString(action.Label)
	savePath := toNullString(action.SavePath)

	limitDL := toNullInt64(action.LimitDownloadSpeed)
	limitUL := toNullInt64(action.LimitUploadSpeed)
	clientID := toNullInt32(action.ClientID)
	filterID := toNullInt32(int32(action.FilterID))

	var err error
	if action.ID != 0 {
		log.Info().Msg("UPDATE existing record")
		_, err = r.db.Exec(`UPDATE action SET name = ?, type = ?, enabled = ?, exec_cmd = ?, exec_args = ?, watch_folder = ? , category =? , tags = ?, label = ?, save_path = ?, paused = ?, ignore_rules = ?, limit_upload_speed = ?, limit_download_speed = ?, client_id = ? 
			 WHERE id = ?`, action.Name, action.Type, action.Enabled, execCmd, execArgs, watchFolder, category, tags, label, savePath, action.Paused, action.IgnoreRules, limitUL, limitDL, clientID, action.ID)
	} else {
		var res sql.Result

		res, err = r.db.Exec(`INSERT INTO action(name, type, enabled, exec_cmd, exec_args, watch_folder, category, tags, label, save_path, paused, ignore_rules, limit_upload_speed, limit_download_speed, client_id, filter_id)
			VALUES (?, ?, ?, ?, ?,? ,?, ?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`, action.Name, action.Type, action.Enabled, execCmd, execArgs, watchFolder, category, tags, label, savePath, action.Paused, action.IgnoreRules, limitUL, limitDL, clientID, filterID)
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}

		resId, _ := res.LastInsertId()
		log.Info().Msgf("LAST INSERT ID %v", resId)
		action.ID = int(resId)
	}

	return &action, nil
}

func (r *ActionRepo) ToggleEnabled(actionID int) error {

	var err error
	var res sql.Result

	res, err = r.db.Exec(`UPDATE action SET enabled = NOT enabled WHERE id = ?`, actionID)
	if err != nil {
		log.Error().Err(err)
		return err
	}

	resId, _ := res.LastInsertId()
	log.Info().Msgf("LAST INSERT ID %v", resId)

	return nil
}

func toNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

func toNullInt32(s int32) sql.NullInt32 {
	return sql.NullInt32{
		Int32: s,
		Valid: s != 0,
	}
}
func toNullInt64(s int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: s,
		Valid: s != 0,
	}
}
