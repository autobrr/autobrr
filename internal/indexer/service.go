// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sanitize"

	"github.com/gosimple/slug"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

type releaseRepo interface {
	UpdateBaseURL(ctx context.Context, indexer string, oldBaseURL, newBaseURL string) error
}

type indexerRepo interface {
	Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error)
	Update(ctx context.Context, indexer *domain.Indexer) error
	List(ctx context.Context) ([]domain.Indexer, error)
	Delete(ctx context.Context, id int) error
	DeleteArchived(ctx context.Context, id int) error
	FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error)
	FindByID(ctx context.Context, id int) (*domain.Indexer, error)
	GetBy(ctx context.Context, req domain.GetIndexerRequest) (*domain.Indexer, error)
	ToggleEnabled(ctx context.Context, indexerID int, enabled bool) error
	ReconcileDeprecations(ctx context.Context, deprecations []domain.IndexerDeprecation, activeIdentifiers map[string]struct{}) error
	ListDeprecations(ctx context.Context) ([]domain.IndexerDeprecation, error)
}

type eventBus interface {
	EmitIndexerDeleted(event events.IndexerChangeEvent)
	EmitIndexerToggled(event events.IndexerChangeEvent)
}

type Service struct {
	log         zerolog.Logger
	eventBus    eventBus
	config      *domain.Config
	repo        indexerRepo
	releaseRepo releaseRepo
	ApiService  apiService

	// contains all raw indexer definitions
	definitions map[string]domain.IndexerDefinition
	// definition with indexer data
	mappedDefinitions map[string]*domain.IndexerDefinition
	// map server:channel:announce to indexer.Identifier
	lookupIRCServerDefinition map[string]map[string]*domain.IndexerDefinition
}

func NewService(log zerolog.Logger, bus eventBus, config *domain.Config, repo indexerRepo, releaseRepo releaseRepo, apiService apiService) *Service {
	return &Service{
		log:                       log.With().Str("module", "indexer").Logger(),
		eventBus:                  bus,
		config:                    config,
		repo:                      repo,
		releaseRepo:               releaseRepo,
		ApiService:                apiService,
		lookupIRCServerDefinition: make(map[string]map[string]*domain.IndexerDefinition),
		definitions:               make(map[string]domain.IndexerDefinition),
		mappedDefinitions:         make(map[string]*domain.IndexerDefinition),
	}
}

func (s *Service) Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	// sanitize user input
	indexer.Name = sanitize.String(indexer.Name)

	for key, val := range indexer.Settings {
		indexer.Settings[key] = sanitize.String(val)
	}

	// if indexer is rss or torznab do additional cleanup for identifier
	if indexer.ImplementationIsFeed() {
		// make lowercase
		cleanName := strings.ToLower(indexer.Name)

		// torznab-name OR rss-name
		indexer.Identifier = slug.Make(fmt.Sprintf("%s-%s", indexer.Implementation, cleanName))
	}

	if indexer.IdentifierExternal == "" {
		indexer.IdentifierExternal = indexer.Name
	}

	i, err := s.repo.Store(ctx, indexer)
	if err != nil {
		s.log.Error().Err(err).Interface("indexer", indexer).Msg("failed to store indexer")
		return nil, err
	}

	// add to indexerInstances
	if err = s.addIndexer(*i); err != nil {
		s.log.Error().Err(err).Str("indexer", indexer.Name).Msg("failed to add indexer")
		return nil, err
	}

	return i, nil
}

func (s *Service) Update(ctx context.Context, indexer *domain.Indexer) error {
	currentIndexer, err := s.repo.FindByID(ctx, int(indexer.ID))
	if err != nil {
		return errors.Wrap(err, "could not find indexer by id: %v", indexer.ID)
	}
	if currentIndexer.Archived {
		return domain.ErrIndexerArchived
	}

	// sanitize user input
	indexer.Name = sanitize.String(indexer.Name)

	for key, val := range indexer.Settings {
		if domain.IsRedactedString(val) {
			currentVal, ok := currentIndexer.Settings[key]
			if !ok {
				return errors.New("could not find setting in current indexer")
			}
			//indexer.Settings[key] = sanitize.String(currentVal)
			indexer.Settings[key] = currentVal
			continue
		}

		indexer.Settings[key] = sanitize.String(val)
	}

	// settings are persisted wholesale, so an update that omits a saved credential would
	// silently delete it; require the client to echo it back, redacted or not
	for key, val := range currentIndexer.Settings {
		if val == "" || !domain.IsSecretIndexerSetting(key) {
			continue
		}

		if _, ok := indexer.Settings[key]; !ok {
			return errors.New("update omits saved secret setting '%s'", key)
		}
	}

	// only IRC indexers have baseURL set
	if indexer.Implementation == domain.IndexerImplementationIRC {
		if indexer.BaseURL == "" {
			return errors.New("indexer baseURL must not be empty")
		}

		// check if baseURL has been updated and update releases if it was
		if currentIndexer.BaseURL != indexer.BaseURL {

			// update urls of releases
			err = s.releaseRepo.UpdateBaseURL(ctx, indexer.Identifier, currentIndexer.BaseURL, indexer.BaseURL)
			if err != nil {
				return errors.Wrap(err, "could not update release urls with new baseURL: %s", indexer.BaseURL)
			}
		}
	}

	if err := s.repo.Update(ctx, indexer); err != nil {
		s.log.Error().Err(err).Interface("indexer", indexer).Msg("could not update indexer")
		return err
	}

	// add to indexerInstances
	if err = s.updateIndexer(indexer); err != nil {
		s.log.Error().Err(err).Str("indexer", indexer.Name).Msg("failed to add indexer")
		return err
	}

	// always publish: a changed-gate computed from the pre-write snapshot misses racing
	// opposite toggles, and the handler reconciles idempotently against persisted state.
	// Publish the stored indexer, not the update payload, so the handler sees a populated
	// Implementation regardless of what the request carried
	if currentIndexer.ImplementationIsFeed() {
		toggled := *currentIndexer
		toggled.Enabled = indexer.Enabled

		s.eventBus.EmitIndexerToggled(events.IndexerChangeEvent{
			Event:   events.Event{Type: events.IndexerToggleEnabled},
			Indexer: &toggled,
		})
	}

	s.log.Debug().Str("indexer", indexer.Name).Msg("successfully updated indexer")

	return nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	indexer, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if indexer.Archived {
		return domain.ErrIndexerArchived
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.log.Error().Err(err).Int("indexer_id", id).Msg("could not delete indexer")
		return err
	}

	// remove from lookup tables
	s.removeIndexer(*indexer)

	if err := s.ApiService.RemoveClient(indexer.Identifier); err != nil {
		s.log.Error().Err(err).Str("indexer", indexer.Name).Msg("could not delete indexer api client")
	}

	s.eventBus.EmitIndexerDeleted(events.IndexerChangeEvent{
		Event:   events.Event{Type: events.IndexerDeleted},
		Indexer: indexer,
	})

	return nil
}

// DeleteArchived permanently removes an archived indexer whose filter links were pruned.
func (s *Service) DeleteArchived(ctx context.Context, id int) error {
	if err := s.repo.DeleteArchived(ctx, id); err != nil {
		s.log.Error().Err(err).Int("indexer_id", id).Msg("could not delete archived indexer")
		return err
	}

	return nil
}

func (s *Service) FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error) {
	indexers, err := s.repo.FindByFilterID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", id).Msg("could not find indexers by filter id")
		return nil, err
	}

	return indexers, nil
}

func (s *Service) FindByID(ctx context.Context, id int) (*domain.Indexer, error) {
	indexer, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Int("indexer_id", id).Msg("could not find indexer by id")
		return nil, err
	}

	return indexer, nil
}

func (s *Service) List(ctx context.Context) ([]domain.Indexer, error) {
	indexers, err := s.repo.List(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not get indexer list")
		return nil, err
	}

	return indexers, nil
}

func (s *Service) GetBy(ctx context.Context, req domain.GetIndexerRequest) (*domain.Indexer, error) {
	indexer, err := s.repo.GetBy(ctx, req)
	if err != nil {
		s.log.Error().Err(err).Interface("indexer", req).Msg("could not get indexer")
		return nil, err
	}

	return indexer, nil
}

func (s *Service) GetAll() ([]*domain.IndexerDefinition, error) {
	var res = make([]*domain.IndexerDefinition, 0)

	for _, indexer := range s.mappedDefinitions {
		if indexer == nil {
			continue
		}

		res = append(res, indexer)
	}

	// sort by name
	sort.SliceStable(res, func(i, j int) bool {
		return strings.ToLower(res[i].Name) < strings.ToLower(res[j].Name)
	})

	return res, nil
}

func (s *Service) mapIndexers() (map[string]*domain.IndexerDefinition, error) {
	indexers, err := s.repo.List(context.Background())
	if err != nil {
		s.log.Error().Err(err).Msg("could not read indexer list")
		return nil, err
	}

	for _, indexer := range indexers {
		indexerDefinition, err := s.mapIndexer(indexer)
		if err != nil {
			continue
		}

		if indexerDefinition == nil {
			continue
		}

		s.mappedDefinitions[indexer.Identifier] = indexerDefinition
	}

	return s.mappedDefinitions, nil
}

func (s *Service) mapIndexer(indexer domain.Indexer) (*domain.IndexerDefinition, error) {
	definitionName := indexer.Identifier

	if indexer.ImplementationIsFeed() {
		definitionName = string(indexer.Implementation)
	}

	d, ok := s.getDefinitionByName(definitionName)
	if !ok {
		// if no indexerDefinition found, continue
		return nil, domain.ErrIndexerNotFound
	}

	d.ID = int(indexer.ID)
	d.Name = indexer.Name
	d.Identifier = indexer.Identifier
	d.IdentifierExternal = indexer.IdentifierExternal
	d.Implementation = indexer.Implementation
	d.BaseURL = indexer.BaseURL
	d.Enabled = indexer.Enabled

	d.UseProxy = indexer.UseProxy
	d.ProxyID = indexer.ProxyID

	if d.SettingsMap == nil {
		d.SettingsMap = make(map[string]string)
	}

	if d.Implementation == domain.IndexerImplementationLegacy {
		d.Implementation = domain.IndexerImplementationIRC
	}

	// map settings
	// add value to settings objects
	for i, setting := range d.Settings {
		if v, ok := indexer.Settings[setting.Name]; ok {
			setting.Value = v

			d.SettingsMap[setting.Name] = v
		}

		d.Settings[i] = setting
	}

	return d, nil
}

func (s *Service) updateMapIndexer(indexer *domain.Indexer) (*domain.IndexerDefinition, error) {
	d, ok := s.mappedDefinitions[indexer.Identifier]
	if !ok {
		return nil, domain.ErrIndexerNotFound
	}

	d.ID = int(indexer.ID)
	d.Name = indexer.Name
	d.Identifier = indexer.Identifier
	d.IdentifierExternal = indexer.IdentifierExternal
	d.Implementation = indexer.Implementation
	d.BaseURL = indexer.BaseURL
	d.Enabled = indexer.Enabled

	d.UseProxy = indexer.UseProxy
	d.ProxyID = indexer.ProxyID

	if d.SettingsMap == nil {
		d.SettingsMap = make(map[string]string)
	}

	if d.Implementation == domain.IndexerImplementationLegacy {
		d.Implementation = domain.IndexerImplementationIRC
	}

	// map settings
	// add value to settings objects
	for i, setting := range d.Settings {
		if v, ok := indexer.Settings[setting.Name]; ok {
			setting.Value = v

			d.SettingsMap[setting.Name] = v
		}

		d.Settings[i] = setting
	}

	return d, nil
}

func (s *Service) GetTemplates() ([]domain.IndexerDefinition, error) {
	definitions := s.definitions

	ret := make([]domain.IndexerDefinition, 0)
	for _, definition := range definitions {
		ret = append(ret, definition)
	}

	// sort by name
	sort.SliceStable(ret, func(i, j int) bool {
		return strings.ToLower(ret[i].Name) < strings.ToLower(ret[j].Name)
	})

	return ret, nil
}

// ListDeprecations returns the known indexer deprecations (removed/retired indexers) so the
// UI can surface friendly names, reasons and cleanup affordances.
func (s *Service) ListDeprecations(ctx context.Context) ([]domain.IndexerDeprecation, error) {
	return s.repo.ListDeprecations(ctx)
}

// reconcileDeprecations projects bundled tombstones into the database before runtime indexers
// are mapped. A custom or bundled active definition with the same identifier revives the row.
func (s *Service) reconcileDeprecations(ctx context.Context, deprecations []domain.IndexerDeprecation) error {
	registered := make(map[string]struct{}, len(deprecations))
	for _, dep := range deprecations {
		registered[dep.Identifier] = struct{}{}
	}

	indexers, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	activeIdentifiers := make(map[string]struct{}, len(indexers))
	var untracked []string
	for _, indexer := range indexers {
		definitionName := indexer.Identifier
		if indexer.ImplementationIsFeed() {
			definitionName = string(indexer.Implementation)
		}

		if _, live := s.definitions[definitionName]; live {
			activeIdentifiers[indexer.Identifier] = struct{}{}
			continue
		}
		if _, known := registered[indexer.Identifier]; known {
			continue
		}
		untracked = append(untracked, indexer.Identifier)
	}

	if err := s.repo.ReconcileDeprecations(ctx, deprecations, activeIdentifiers); err != nil {
		return err
	}

	if len(untracked) > 0 {
		sort.Strings(untracked)
		s.log.Warn().Msgf("found %d indexer(s) with no active definition or bundled tombstone: %s - restore a custom definition or report the missing tombstone", len(untracked), strings.Join(untracked, ", "))
	}

	return nil
}

func (s *Service) Start() error {
	// load all indexer definitions
	if err := s.LoadIndexerDefinitions(); err != nil {
		s.log.Error().Err(err).Msg("could not load indexer definitions")
		return err
	}

	if s.config.CustomDefinitions != "" {
		// load custom indexer definitions
		if err := s.LoadCustomIndexerDefinitions(); err != nil {
			return errors.Wrap(err, "could not load custom indexer definitions")
		}
	}

	deprecations, err := LoadDeprecatedIndexerDefinitions()
	if err != nil {
		return err
	}

	if err := s.reconcileDeprecations(context.Background(), deprecations); err != nil {
		return errors.Wrap(err, "could not reconcile indexer deprecations")
	}

	// load the indexers' setup by the user
	indexerDefinitions, err := s.mapIndexers()
	if err != nil {
		return err
	}

	for _, indexer := range indexerDefinitions {
		switch indexer.Implementation {
		case domain.IndexerImplementationIRC:
			// add to irc server lookup table
			s.mapIRCServerDefinitionLookup(indexer.IRC.Server, indexer)

			// check if it has api and add to api service
			if indexer.Enabled && indexer.HasApi() {
				if err := s.ApiService.AddClient(indexer.Identifier, indexer.SettingsMap, indexer.ProxyID, indexer.UseProxy); err != nil {
					s.log.Error().Stack().Err(err).Str("indexer", indexer.Identifier).Msg("indexer.start: could not init indexer api client")
				}
			}
		}
	}

	s.log.Info().Int("count", len(indexerDefinitions)).Msg("Loaded indexers")

	return nil
}

func (s *Service) removeIndexer(indexer domain.Indexer) {
	// remove mapped definition
	delete(s.mappedDefinitions, indexer.Identifier)
}

func (s *Service) addIndexer(indexer domain.Indexer) error {
	indexerDefinition, err := s.mapIndexer(indexer)
	if err != nil {
		return err
	}

	if indexerDefinition == nil {
		return errors.New("addindexer: could not find definition")
	}

	switch indexer.Implementation {
	case domain.IndexerImplementationIRC:
		// add to irc server lookup table
		s.mapIRCServerDefinitionLookup(indexerDefinition.IRC.Server, indexerDefinition)

		// check if it has api and add to api service
		if indexerDefinition.HasApi() {
			if err := s.ApiService.AddClient(indexerDefinition.Identifier, indexerDefinition.SettingsMap, indexerDefinition.ProxyID, indexerDefinition.UseProxy); err != nil {
				s.log.Error().Stack().Err(err).Str("indexer", indexer.Identifier).Msg("indexer.addIndexer: could not init indexer api client")
			}
		}
	}

	s.mappedDefinitions[indexer.Identifier] = indexerDefinition

	return nil
}

func (s *Service) updateIndexer(indexer *domain.Indexer) error {
	indexerDefinition, err := s.updateMapIndexer(indexer)
	if err != nil {
		return err
	}

	if indexerDefinition == nil {
		return errors.New("update indexer: could not find definition")
	}

	switch indexer.Implementation {
	case domain.IndexerImplementationIRC:
		// add to irc server lookup table
		s.mapIRCServerDefinitionLookup(indexerDefinition.IRC.Server, indexerDefinition)

		// check if it has api and add to api service
		if indexerDefinition.HasApi() {
			if err := s.ApiService.AddClient(indexerDefinition.Identifier, indexerDefinition.SettingsMap, indexerDefinition.ProxyID, indexerDefinition.UseProxy); err != nil {
				s.log.Error().Stack().Err(err).Str("indexer", indexer.Identifier).Msg("indexer.updateIndexer: could not init indexer api client")
			}
		}
	}

	s.mappedDefinitions[indexer.Identifier] = indexerDefinition

	return nil
}

// mapIRCServerDefinitionLookup map irc stuff to indexer.name
// map[irc.network.test][indexer1] = indexer1
// map[irc.network.test][indexer2] = indexer2
func (s *Service) mapIRCServerDefinitionLookup(ircServer string, indexerDefinition *domain.IndexerDefinition) {
	if indexerDefinition.IRC != nil {
		// check if already exists, if ok add it to existing, otherwise create new
		_, exists := s.lookupIRCServerDefinition[ircServer]
		if !exists {
			s.lookupIRCServerDefinition[ircServer] = map[string]*domain.IndexerDefinition{}
		}

		s.lookupIRCServerDefinition[ircServer][indexerDefinition.Identifier] = indexerDefinition
	}
}

// LoadIndexerDefinitions load definitions from golang embed fs
func (s *Service) LoadIndexerDefinitions() error {
	entries, err := fs.ReadDir(Definitions, "definitions")
	if err != nil {
		return errors.Wrap(err, "could not read indexer definitions directory")
	}

	if len(entries) == 0 {
		return errors.Wrap(err, "could not read directory")
	}

	for _, entry := range entries {
		fileExtension := filepath.Ext(entry.Name())
		if fileExtension != ".yaml" {
			continue
		}

		file := "definitions/" + entry.Name()

		s.log.Trace().Str("file", file).Msg("parsing indexer definition")

		data, err := fs.ReadFile(Definitions, file)
		if err != nil {
			return errors.Wrap(err, "could not read indexer definition file: %s", file)
		}

		var d domain.IndexerDefinition
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)

		if err = dec.Decode(&d); err != nil {
			return errors.Wrap(err, "could not unmarshal indexer definition file: %s", file)
		}
		if err = d.ValidateIRCAuth(); err != nil {
			return errors.Wrap(err, "invalid indexer definition file: %s", file)
		}

		d.Prepare()

		s.definitions[d.Identifier] = d
	}

	s.log.Debug().Int("count", len(s.definitions)).Msg("loaded indexer definitions")

	return nil
}

var ErrIndexerDefinitionDeprecated = errors.New("DEPRECATED: indexer definition version")

func isValidExtension(ext string) bool {
	return ext == ".yaml" || ext == ".yml"
}

func OpenAndProcessDefinition(file string) (*domain.IndexerDefinition, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "could not open file: %s", file)
	}

	// peek at the version field to decide which schema to decode into
	var meta struct {
		Version int `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, errors.Wrap(err, "could not detect definition version: %s", file)
	}

	// version 2+ maps directly onto the current IndexerDefinition schema
	if meta.Version >= 2 {
		var d domain.IndexerDefinition

		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(false)

		if err := dec.Decode(&d); err != nil {
			return nil, errors.Wrap(err, "could not decode definition file: %s", file)
		}
		if err := d.ValidateIRCAuth(); err != nil {
			return nil, errors.Wrap(err, "invalid definition file: %s", file)
		}

		d.Prepare()

		return &d, nil
	}

	// legacy (v1) definitions use the compatibility struct and get converted
	var d *domain.IndexerDefinitionCustom

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(false)

	if err := dec.Decode(&d); err != nil {
		return nil, errors.Wrap(err, "could not decode definition file: %s", file)
	}

	if d == nil {
		return nil, errors.New("empty definition file")
	}

	if d.Implementation == domain.IndexerImplementationLegacy {
		d.Implementation = domain.IndexerImplementationIRC
	}

	definition := d.ToIndexerDefinition()
	if err := definition.ValidateIRCAuth(); err != nil {
		return nil, errors.Wrap(err, "invalid definition file: %s", file)
	}

	return definition, nil
}

func OpenAndDecodeDefinition(file string, data any) error {
	f, err := os.Open(file)
	if err != nil {
		return errors.Wrap(err, "could not open file: %s", file)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(false)

	if err = dec.Decode(data); err != nil {
		return errors.Wrap(err, "could not decode definition file: %s", file)
	}

	if data == nil {
		return errors.New("empty definition file")
	}

	return nil
}

// LoadCustomIndexerDefinitions load definitions from custom path
func (s *Service) LoadCustomIndexerDefinitions() error {
	if s.config.CustomDefinitions == "" {
		return nil
	}

	outputDirRead, err := os.Open(s.config.CustomDefinitions)
	if err != nil {
		s.log.Error().Err(err).Str("custom_definitions_path", s.config.CustomDefinitions).Msg("failed opening custom definitions directory")
		return errors.Wrap(err, "could not open custom definitions directory: %s", s.config.CustomDefinitions)
	}

	defer outputDirRead.Close()

	entries, err := outputDirRead.ReadDir(0)
	if err != nil {
		return errors.Wrap(err, "could not read customDefinitions directory: %s", s.config.CustomDefinitions)
	}

	customCount := 0

	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if !isValidExtension(ext) {
			s.log.Warn().Str("ext", ext).Str("file", entry.Name()).Msg("unsupported extension for definition file")
			continue
		}

		file := filepath.Join(s.config.CustomDefinitions, entry.Name())

		s.log.Trace().Str("file", file).Msg("parsing custom definition")

		definition, err := OpenAndProcessDefinition(file)
		if err != nil {
			s.log.Error().Err(err).Str("file", file).Msg("could not open definition file")
			continue
		}

		s.definitions[definition.Identifier] = *definition

		customCount++
	}

	s.log.Debug().Int("count", customCount).Msg("Loaded custom indexer definitions")

	return nil
}

func (s *Service) GetIndexersByIRCNetwork(server string) []*domain.IndexerDefinition {
	server = strings.ToLower(server)

	var indexerDefinitions []*domain.IndexerDefinition

	// get indexer definitions matching irc network from lookup table
	if srv, idOk := s.lookupIRCServerDefinition[server]; idOk {
		for _, definition := range srv {
			indexerDefinitions = append(indexerDefinitions, definition)
		}
	}

	return indexerDefinitions
}

//func (s *service) GetTorznabIndexers() []domain.IndexerDefinition {
//	indexerDefinitions := make([]domain.IndexerDefinition, 0)
//
//	for _, definition := range s.torznabIndexers {
//		if definition != nil {
//			indexerDefinitions = append(indexerDefinitions, *definition)
//		}
//	}
//
//	return indexerDefinitions
//}
//
//func (s *service) GetRSSIndexers() []domain.IndexerDefinition {
//	indexerDefinitions := make([]domain.IndexerDefinition, 0)
//
//	for _, definition := range s.rssIndexers {
//		if definition != nil {
//			indexerDefinitions = append(indexerDefinitions, *definition)
//		}
//	}
//
//	return indexerDefinitions
//}

func (s *Service) getDefinitionByName(name string) (*domain.IndexerDefinition, bool) {
	if v, ok := s.definitions[name]; ok {
		return &v, true
	}

	return nil, false
}

func (s *Service) GetMappedDefinitionByName(name string) (*domain.IndexerDefinition, bool) {
	v, ok := s.mappedDefinitions[name]
	if !ok {
		return nil, false
	}

	return v, true
}

func (s *Service) TestApi(ctx context.Context, req domain.IndexerTestApiRequest) error {
	indexer, err := s.FindByID(ctx, req.IndexerId)
	if err != nil {
		return err
	}
	if indexer.Archived {
		return domain.ErrIndexerArchived
	}

	def, ok := s.GetMappedDefinitionByName(indexer.Identifier)
	if !ok {
		return errors.New("could not find indexer definition: %s", indexer.Identifier)
	}

	if !def.HasApi() {
		return errors.New("indexer (%s) does not support api", indexer.Identifier)
	}

	if domain.IsRedactedString(req.ApiKey) {
		apikey, ok := indexer.Settings["api_key"]
		if !ok {
			return errors.New("could not find apikey in indexer settings")
		}

		req.ApiKey = apikey
	}

	req.Identifier = def.Identifier
	req.ProxyID = def.ProxyID
	req.UseProxy = def.UseProxy

	if _, err = s.ApiService.TestConnection(ctx, req); err != nil {
		s.log.Error().Err(err).Str("indexer", indexer.Identifier).Msg("error testing indexer api")
		return err
	}

	s.log.Info().Str("indexer", indexer.Identifier).Msg("indexer api test successful!")

	return nil
}

func (s *Service) ToggleEnabled(ctx context.Context, indexerID int, enabled bool) error {
	indexer, err := s.FindByID(ctx, indexerID)
	if err != nil {
		return err
	}
	if indexer.Archived {
		return domain.ErrIndexerArchived
	}

	if err := s.repo.ToggleEnabled(ctx, int(indexer.ID), enabled); err != nil {
		s.log.Error().Err(err).Msg("could not update indexer enabled")
		return err
	}

	indexer.Enabled = enabled

	// update indexerInstances
	if err := s.updateIndexer(indexer); err != nil {
		s.log.Error().Err(err).Str("indexer", indexer.Name).Msg("failed to update indexer")
		return err
	}

	// feed jobs are stopped and started by the feed service via event because the feed service
	// can't be imported here. Always published: a changed-gate computed from the pre-write
	// snapshot misses racing opposite toggles, and the handler reconciles idempotently
	if indexer.ImplementationIsFeed() {
		s.eventBus.EmitIndexerToggled(events.IndexerChangeEvent{
			Event:   events.Event{Type: events.IndexerToggleEnabled},
			Indexer: indexer,
		})
	}

	s.log.Debug().Str("indexer", indexer.Name).Int("indexer_id", indexerID).Bool("enabled", enabled).Msg("indexer.toggleEnabled: update indexer state")

	return nil
}
