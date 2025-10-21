// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import (
	"database/sql"

	"github.com/autobrr/autobrr/pkg/migrator"

	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/rs/zerolog"
)

func PostgresMigrations(db *sql.DB, logger zerolog.Logger) *migrator.Migrator {
	migrate := migrator.NewMigrate(
		db,
		migrator.WithEngine(migrator.EnginePostgres),
		migrator.WithEmbedFS(SchemaMigrationsPostgres, "postgres"),
		migrator.WithSchemaFile("current_schema_postgres.sql"),
		migrator.WithLogger(zstdlog.NewStdLoggerWithLevel(logger.With().Str("module", "database-migrations").Logger(), zerolog.InfoLevel)),
	)

	migrate.AddFileMigration("0_base_schema_postgres.sql")
	migrate.AddFileMigration("1_create_notifications_table.sql")
	migrate.AddFileMigration("2_create_feed_and_feed_cache_tables.sql")
	migrate.AddFileMigration("3_add_indexer_implementation_column.sql")
	migrate.AddFileMigration("4_clean_release_columns_and_add_origins_to_filter.sql")
	migrate.AddFileMigration("5_add_match_except_other_filter_columns.sql")
	migrate.AddFileMigration("6_rename_group_back_to_release_group.sql")
	migrate.AddFileMigration("7_add_action_reannounce_columns.sql")
	migrate.AddFileMigration("8_add_action_limit_ratio_and_seed_time.sql")
	migrate.AddFileMigration("9_add_filter_max_downloads.sql")
	migrate.AddFileMigration("10_create_database_indexes.sql")
	migrate.AddFileMigration("11_add_client_and_filter_columns_to_release_action_status.sql")
	migrate.AddFileMigration("12_add_external_script_and_webhook_filter_columns.sql")
	migrate.AddFileMigration("13_add_action_skip_hash_check_and_content_layout.sql")
	migrate.AddFileMigration("14_add_filter_except_origins.sql")
	migrate.AddFileMigration("15_create_api_key_table.sql")
	migrate.AddFileMigration("16_add_feed_timeout.sql")
	migrate.AddFileMigration("17_add_feed_max_age_last_run_and_cookie.sql")
	migrate.AddFileMigration("18_add_filter_release_tags_matching.sql")
	migrate.AddFileMigration("19_irc_network_add_sasl_auth.sql")
	migrate.AddFileMigration("20_add_indexer_base_url.sql")
	migrate.AddFileMigration("21_add_filter_smart_episode.sql")
	migrate.AddFileMigration("22_add_filter_language_matching.sql")
	migrate.AddFileMigration("23_release_action_status_add_filter_id.sql")
	migrate.AddFileMigration("24_add_release_info_and_download_urls.sql")
	migrate.AddFileMigration("25_add_filter_tags_match_logic_and_set_defaults.sql")
	migrate.AddFileMigration("26_add_notification_priority.sql")
	migrate.AddFileMigration("27_add_notification_topic.sql")
	migrate.AddFileMigration("28_add_filter_description_matching.sql")
	migrate.AddFileMigration("29_add_release_action_status_action_id.sql")
	migrate.AddFileMigration("30_add_irc_bouncer_support.sql")
	migrate.AddFileMigration("31_update_release_action_status_constraint.sql")
	migrate.AddFileMigration("32_create_filter_external.sql")
	migrate.AddFileMigration("33_rebuild_feed_cache_with_feed_id_foreign_key.sql")
	migrate.AddFileMigration("34_add_action_external_client_id.sql")
	migrate.AddFileMigration("35_add_filter_external_webhook_retry_columns.sql")
	migrate.AddFileMigration("36_rename_external_webhook_retry_columns.sql")
	migrate.AddFileMigration("37_remove_webhook_retry_max_jitter_seconds.sql")
	migrate.AddFileMigration("38_add_irc_network_bot_mode.sql")
	migrate.AddFileMigration("39_add_feed_max_age.sql")
	migrate.AddFileMigration("40_add_action_priority.sql")
	migrate.AddFileMigration("41_add_action_external_client.sql")
	migrate.AddFileMigration("42_add_filter_seeders_leechers_limits.sql")
	migrate.AddFileMigration("43_update_nebulance_irc_server.sql")
	migrate.AddFileMigration("44_release_change_timestamp_type.sql")
	migrate.AddFileMigration("45_update_animebytes_irc_server_and_name.sql")
	migrate.AddFileMigration("46_add_action_first_last_piece_priority.sql")
	migrate.AddFileMigration("47_add_indexer_identifier_external_and_populate.sql")
	migrate.AddFileMigration("48_add_release_and_filter_month_day_columns.sql")
	migrate.AddFileMigration("49_create_proxy_table_and_add_proxy_support.sql")
	migrate.AddFileMigration("50_update_fuzer_indexer_base_url.sql")
	migrate.AddFileMigration("51_create_filter_and_database_indexes.sql")
	migrate.AddFileMigration("52_update_fuzer_irc_server.sql")
	migrate.AddFileMigration("53_update_multiple_irc_servers.sql")
	migrate.AddFileMigration("54_update_redacted_indexer_base_url.sql")
	migrate.AddFileMigration("55_update_seedpool_irc_port_and_tls.sql")
	migrate.AddFileMigration("56_add_announce_type_support.sql")
	migrate.AddFileMigration("57_create_list_tables.sql")
	migrate.AddFileMigration("58_add_filter_record_label_matching.sql")
	migrate.AddFileMigration("59_create_release_profile_duplicate_and_add_release_fields.sql")
	migrate.AddFileMigration("60_update_ptp_announce_channel_name.sql")
	migrate.AddFileMigration("61_set_default_announce_types_for_filters.sql")
	migrate.AddFileMigration("62_add_list_skip_clean_sanitize.sql")
	migrate.AddFileMigration("63_update_rocket_hd_auth_and_channel_passwords.sql")
	migrate.AddFileMigration("64_create_sessions_table.sql")
	migrate.AddFileMigration("65_migrate_ulcx_network.sql")
	migrate.AddFileMigration("66_fix_macro_time.sql")
	migrate.AddFileMigration("67_action_add_download_path.sql")
	migrate.AddFileMigration("68_list_include_year.sql")
	migrate.AddFileMigration("69_filter_external_on_error.sql")
	migrate.AddFileMigration("70_filter_notifications.sql")
	migrate.AddFileMigration("71_indexers_update_revtt_domain.sql")
	migrate.AddFileMigration("72_duplicate_profiles_add_hybrid.sql")
	migrate.AddFileMigration("73_indexers_update_reelflix_domain.sql")

	return migrate
}
