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
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sanitize"

	"github.com/asaskevich/EventBus"
	"github.com/gosimple/slug"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

type Service interface {
	Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error)
	Update(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error)
	Delete(ctx context.Context, id int) error
	FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error)
	FindByID(ctx context.Context, id int) (*domain.Indexer, error)
	List(ctx context.Context) ([]domain.Indexer, error)
	GetBy(ctx context.Context, req domain.GetIndexerRequest) (*domain.Indexer, error)
	GetAll() ([]*domain.IndexerDefinition, error)
	GetTemplates() ([]domain.IndexerDefinition, error)
	LoadIndexerDefinitions() error
	GetIndexersByIRCNetwork(server string) []*domain.IndexerDefinition
	GetTorznabIndexers() []domain.IndexerDefinition
	GetMappedDefinitionByName(name string) (*domain.IndexerDefinition, error)
	Start() error
	TestApi(ctx context.Context, req domain.IndexerTestApiRequest) error
	ToggleEnabled(ctx context.Context, indexerID int, enabled bool) error
}

type service struct {
	log         zerolog.Logger
	config      *domain.Config
	repo        domain.IndexerRepo
	releaseRepo domain.ReleaseRepo
	ApiService  APIService
	scheduler   scheduler.Service
	bus         EventBus.Bus

	// contains all raw indexer definitions
	definitions map[string]domain.IndexerDefinition
	// definition with indexer data
	mappedDefinitions map[string]*domain.IndexerDefinition
	// map server:channel:announce to indexer.Identifier
	lookupIRCServerDefinition map[string]map[string]*domain.IndexerDefinition
	// torznab indexers
	torznabIndexers map[string]*domain.IndexerDefinition
	// newznab indexers
	newznabIndexers map[string]*domain.IndexerDefinition
	// rss indexers
	rssIndexers map[string]*domain.IndexerDefinition
}

func NewService(log logger.Logger, config *domain.Config, bus EventBus.Bus, repo domain.IndexerRepo, releaseRepo domain.ReleaseRepo, apiService APIService, scheduler scheduler.Service) Service {
	return &service{
		log:                       log.With().Str("module", "indexer").Logger(),
		config:                    config,
		repo:                      repo,
		releaseRepo:               releaseRepo,
		ApiService:                apiService,
		scheduler:                 scheduler,
		bus:                       bus,
		lookupIRCServerDefinition: make(map[string]map[string]*domain.IndexerDefinition),
		torznabIndexers:           make(map[string]*domain.IndexerDefinition),
		newznabIndexers:           make(map[string]*domain.IndexerDefinition),
		rssIndexers:               make(map[string]*domain.IndexerDefinition),
		definitions:               make(map[string]domain.IndexerDefinition),
		mappedDefinitions:         make(map[string]*domain.IndexerDefinition),
	}
}

func (s *service) Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	// sanitize user input
	indexer.Name = sanitize.String(indexer.Name)

	for key, val := range indexer.Settings {
		indexer.Settings[key] = sanitize.String(val)
	}

	// if indexer is rss or torznab do additional cleanup for identifier
	if isImplFeed(indexer.Implementation) {
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
		s.log.Error().Err(err).Msgf("failed to store indexer: %s", indexer.Name)
		return nil, err
	}

	// add to indexerInstances
	if err = s.addIndexer(*i); err != nil {
		s.log.Error().Err(err).Msgf("failed to add indexer: %s", indexer.Name)
		return nil, err
	}

	return i, nil
}

func (s *service) Update(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	// sanitize user input
	indexer.Name = sanitize.String(indexer.Name)

	for key, val := range indexer.Settings {
		indexer.Settings[key] = sanitize.String(val)
	}

	currentIndexer, err := s.repo.FindByID(ctx, int(indexer.ID))
	if err != nil {
		return nil, errors.Wrap(err, "could not find indexer by id: %v", indexer.ID)
	}

	// only IRC indexers have baseURL set
	if indexer.Implementation == string(domain.IndexerImplementationIRC) {
		if indexer.BaseURL == "" {
			return nil, errors.New("indexer baseURL must not be empty")
		}

		// check if baseURL has been updated and update releases if it was
		if currentIndexer.BaseURL != indexer.BaseURL {

			// update urls of releases
			err = s.releaseRepo.UpdateBaseURL(ctx, indexer.Identifier, currentIndexer.BaseURL, indexer.BaseURL)
			if err != nil {
				return nil, errors.Wrap(err, "could not update release urls with new baseURL: %s", indexer.BaseURL)
			}
		}
	}

	i, err := s.repo.Update(ctx, indexer)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not update indexer: %+v", indexer)
		return nil, err
	}

	// add to indexerInstances
	if err = s.updateIndexer(*i); err != nil {
		s.log.Error().Err(err).Msgf("failed to add indexer: %s", indexer.Name)
		return nil, err
	}

	if isImplFeed(indexer.Implementation) {
		if currentIndexer.Enabled && !indexer.Enabled {
			s.stopFeed(indexer.Identifier)
		}
	}

	s.log.Debug().Msgf("successfully updated indexer: %s", indexer.Name)

	return i, nil
}

func (s *service) Delete(ctx context.Context, id int) error {
	indexer, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.log.Error().Err(err).Msgf("could not delete indexer by id: %d", id)
		return err
	}

	// remove from lookup tables
	s.removeIndexer(*indexer)

	if err := s.ApiService.RemoveClient(indexer.Identifier); err != nil {
		s.log.Error().Err(err).Msgf("could not delete indexer api client: %s", indexer.Identifier)
	}

	s.bus.Publish(domain.EventIndexerDelete, indexer)

	return nil
}

func (s *service) FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error) {
	indexers, err := s.repo.FindByFilterID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find indexers by filter id: %d", id)
		return nil, err
	}

	return indexers, err
}

func (s *service) FindByID(ctx context.Context, id int) (*domain.Indexer, error) {
	indexers, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find indexer by id: %d", id)
		return nil, err
	}

	return indexers, err
}

func (s *service) List(ctx context.Context) ([]domain.Indexer, error) {
	indexers, err := s.repo.List(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not get indexer list")
		return nil, err
	}

	return indexers, err
}

func (s *service) GetBy(ctx context.Context, req domain.GetIndexerRequest) (*domain.Indexer, error) {
	indexer, err := s.repo.GetBy(ctx, req)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not get indexer by: %v", req)
		return nil, err
	}

	return indexer, err
}

func (s *service) GetAll() ([]*domain.IndexerDefinition, error) {
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

func (s *service) mapIndexers() (map[string]*domain.IndexerDefinition, error) {
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

func (s *service) mapIndexer(indexer domain.Indexer) (*domain.IndexerDefinition, error) {
	definitionName := indexer.Identifier

	if isImplFeed(indexer.Implementation) {
		definitionName = indexer.Implementation
	}

	d := s.getDefinitionByName(definitionName)
	if d == nil {
		// if no indexerDefinition found, continue
		return nil, nil
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

	if d.Implementation == "" {
		d.Implementation = "irc"
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

func (s *service) updateMapIndexer(indexer domain.Indexer) (*domain.IndexerDefinition, error) {
	d, ok := s.mappedDefinitions[indexer.Identifier]
	if !ok {
		return nil, nil
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

	if d.Implementation == "" {
		d.Implementation = "irc"
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

func (s *service) GetTemplates() ([]domain.IndexerDefinition, error) {
	definitions := s.definitions

	ret := make([]domain.IndexerDefinition, 0)
	for _, definition := range definitions {
		ret = append(ret, definition)
	}

	return ret, nil
}

func (s *service) Start() error {
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

	// load the indexers' setup by the user
	indexerDefinitions, err := s.mapIndexers()
	if err != nil {
		return err
	}

	for _, indexer := range indexerDefinitions {
		switch indexer.Implementation {
		case string(domain.IndexerImplementationIRC):
			// add to irc server lookup table
			s.mapIRCServerDefinitionLookup(indexer.IRC.Server, indexer)

			// check if it has api and add to api service
			if indexer.Enabled && indexer.HasApi() {
				if err := s.ApiService.AddClient(indexer.Identifier, indexer.SettingsMap); err != nil {
					s.log.Error().Stack().Err(err).Msgf("indexer.start: could not init api client for: '%s'", indexer.Identifier)
				}
			}

		// handle feeds
		case string(domain.IndexerImplementationRSS):
			s.rssIndexers[indexer.Identifier] = indexer

		case string(domain.IndexerImplementationTorznab):
			s.torznabIndexers[indexer.Identifier] = indexer

		case string(domain.IndexerImplementationNewznab):
			s.newznabIndexers[indexer.Identifier] = indexer
		}
	}

	s.log.Info().Msgf("Loaded %d indexers", len(indexerDefinitions))

	return nil
}

func (s *service) removeIndexer(indexer domain.Indexer) {
	// handle feeds
	switch indexer.Implementation {
	case string(domain.IndexerImplementationRSS):
		delete(s.rssIndexers, indexer.Identifier)

	case string(domain.IndexerImplementationTorznab):
		delete(s.torznabIndexers, indexer.Identifier)

	case string(domain.IndexerImplementationNewznab):
		delete(s.newznabIndexers, indexer.Identifier)
	}

	// remove mapped definition
	delete(s.mappedDefinitions, indexer.Identifier)
}

func (s *service) addIndexer(indexer domain.Indexer) error {
	indexerDefinition, err := s.mapIndexer(indexer)
	if err != nil {
		return err
	}

	if indexerDefinition == nil {
		return errors.New("addindexer: could not find definition")
	}

	switch indexer.Implementation {
	case string(domain.IndexerImplementationIRC):
		// add to irc server lookup table
		s.mapIRCServerDefinitionLookup(indexerDefinition.IRC.Server, indexerDefinition)

		// check if it has api and add to api service
		if indexerDefinition.HasApi() {
			if err := s.ApiService.AddClient(indexerDefinition.Identifier, indexerDefinition.SettingsMap); err != nil {
				s.log.Error().Stack().Err(err).Msgf("indexer.start: could not init api client for: '%s'", indexer.Identifier)
			}
		}

	// handle feeds
	case string(domain.IndexerImplementationRSS):
		s.rssIndexers[indexer.Identifier] = indexerDefinition

	case string(domain.IndexerImplementationTorznab):
		s.torznabIndexers[indexer.Identifier] = indexerDefinition

	case string(domain.IndexerImplementationNewznab):
		s.newznabIndexers[indexer.Identifier] = indexerDefinition
	}

	s.mappedDefinitions[indexer.Identifier] = indexerDefinition

	return nil
}

func (s *service) updateIndexer(indexer domain.Indexer) error {
	indexerDefinition, err := s.updateMapIndexer(indexer)
	if err != nil {
		return err
	}

	if indexerDefinition == nil {
		return errors.New("update indexer: could not find definition")
	}

	switch indexer.Implementation {
	case string(domain.IndexerImplementationIRC):
		// add to irc server lookup table
		s.mapIRCServerDefinitionLookup(indexerDefinition.IRC.Server, indexerDefinition)

		// check if it has api and add to api service
		if indexerDefinition.HasApi() {
			if err := s.ApiService.AddClient(indexerDefinition.Identifier, indexerDefinition.SettingsMap); err != nil {
				s.log.Error().Stack().Err(err).Msgf("indexer.start: could not init api client for: '%s'", indexer.Identifier)
			}
		}

	// handle feeds
	case string(domain.IndexerImplementationRSS):
		s.rssIndexers[indexer.Identifier] = indexerDefinition

	case string(domain.IndexerImplementationTorznab):
		s.torznabIndexers[indexer.Identifier] = indexerDefinition

	case string(domain.IndexerImplementationNewznab):
		s.newznabIndexers[indexer.Identifier] = indexerDefinition
	}

	s.mappedDefinitions[indexer.Identifier] = indexerDefinition

	return nil
}

// mapIRCServerDefinitionLookup map irc stuff to indexer.name
// map[irc.network.test][indexer1] = indexer1
// map[irc.network.test][indexer2] = indexer2
func (s *service) mapIRCServerDefinitionLookup(ircServer string, indexerDefinition *domain.IndexerDefinition) {
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
func (s *service) LoadIndexerDefinitions() error {
	entries, err := fs.ReadDir(Definitions, "definitions")
	if err != nil {
		s.log.Fatal().Err(err).Stack().Msg("failed reading directory")
	}

	if len(entries) == 0 {
		s.log.Fatal().Err(err).Stack().Msg("failed reading directory")
		return errors.Wrap(err, "could not read directory")
	}

	for _, f := range entries {
		fileExtension := filepath.Ext(f.Name())
		if fileExtension != ".yaml" {
			continue
		}

		file := "definitions/" + f.Name()

		s.log.Trace().Msgf("parsing: %s", file)

		data, err := fs.ReadFile(Definitions, file)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("failed reading file: %s", file)
			return errors.Wrap(err, "could not read file: %s", file)
		}

		var d domain.IndexerDefinition
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)

		if err = dec.Decode(&d); err != nil {
			s.log.Error().Stack().Err(err).Msgf("failed unmarshal file: %s", file)
			return errors.Wrap(err, "could not unmarshal file: %s", file)
		}

		if d.Implementation == "" {
			d.Implementation = "irc"
		}

		s.definitions[d.Identifier] = d
	}

	s.log.Debug().Msgf("Loaded %d indexer definitions", len(s.definitions))

	return nil
}

var ErrIndexerDefinitionDeprecated = errors.New("DEPRECATED: indexer definition version")

func isValidExtension(ext string) bool {
	return ext == ".yaml" || ext == ".yml"
}

func OpenAndProcessDefinition(file string) (*domain.IndexerDefinition, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrap(err, "could not open file: %s", file)
	}
	defer f.Close()

	var d *domain.IndexerDefinitionCustom

	dec := yaml.NewDecoder(f)
	dec.KnownFields(false)

	if err = dec.Decode(&d); err != nil {
		return nil, errors.Wrap(err, "could not decode definition file: %s", file)
	}

	if d == nil {
		//s.log.Warn().Msgf("skipping empty file: %s", file)
		return nil, errors.New("empty definition file")
	}

	if d.Implementation == "" {
		d.Implementation = "irc"
	}

	//if d.Implementation == "irc" && d.IRC != nil {
	//	if d.IRC.Parse == nil {
	//		s.log.Warn().Msgf("DEPRECATED: indexer definition version: %s", file)
	//	}
	//}

	return d.ToIndexerDefinition(), nil
}

// LoadCustomIndexerDefinitions load definitions from custom path
func (s *service) LoadCustomIndexerDefinitions() error {
	if s.config.CustomDefinitions == "" {
		return nil
	}

	outputDirRead, err := os.Open(s.config.CustomDefinitions)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed opening custom definitions directory %s", s.config.CustomDefinitions)
		return nil
	}

	defer outputDirRead.Close()

	entries, err := outputDirRead.ReadDir(0)
	if err != nil {
		s.log.Fatal().Err(err).Stack().Msg("failed reading directory")
		return errors.Wrap(err, "could not read directory")
	}

	customCount := 0

	for _, f := range entries {
		ext := filepath.Ext(f.Name())
		if !isValidExtension(ext) {
			s.log.Warn().Msgf("unsupported extension %s, definition file: %s", ext, f.Name())
			continue
		}

		file := filepath.Join(s.config.CustomDefinitions, f.Name())

		s.log.Trace().Msgf("parsing custom definition: %s", file)

		definition, err := OpenAndProcessDefinition(file)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not open definition file: %s", file)
			continue
		}

		s.definitions[definition.Identifier] = *definition

		customCount++
	}

	s.log.Debug().Msgf("Loaded %d custom indexer definitions", customCount)

	return nil
}

func (s *service) GetIndexersByIRCNetwork(server string) []*domain.IndexerDefinition {
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

func (s *service) GetTorznabIndexers() []domain.IndexerDefinition {
	indexerDefinitions := make([]domain.IndexerDefinition, 0)

	for _, definition := range s.torznabIndexers {
		if definition != nil {
			indexerDefinitions = append(indexerDefinitions, *definition)
		}
	}

	return indexerDefinitions
}

func (s *service) GetRSSIndexers() []domain.IndexerDefinition {
	indexerDefinitions := make([]domain.IndexerDefinition, 0)

	for _, definition := range s.rssIndexers {
		if definition != nil {
			indexerDefinitions = append(indexerDefinitions, *definition)
		}
	}

	return indexerDefinitions
}

func (s *service) getDefinitionByName(name string) *domain.IndexerDefinition {
	if v, ok := s.definitions[name]; ok {
		return &v
	}

	return nil
}

func (s *service) GetMappedDefinitionByName(name string) (*domain.IndexerDefinition, error) {
	v, ok := s.mappedDefinitions[name]
	if !ok {
		return nil, errors.New("unknown indexer identifier: %s", name)
	}

	return v, nil
}

func (s *service) getMappedDefinitionByName(name string) *domain.IndexerDefinition {
	if v, ok := s.mappedDefinitions[name]; ok {
		return v
	}

	return nil
}

func (s *service) stopFeed(indexer string) {
	// verify indexer is torznab indexer
	_, ok := s.torznabIndexers[indexer]
	if !ok {
		_, rssOK := s.rssIndexers[indexer]
		if !rssOK {
			return
		}
		return
	}

	if err := s.scheduler.RemoveJobByIdentifier(indexer); err != nil {
		return
	}
}

func (s *service) TestApi(ctx context.Context, req domain.IndexerTestApiRequest) error {
	indexer, err := s.FindByID(ctx, req.IndexerId)
	if err != nil {
		return err
	}

	def := s.getMappedDefinitionByName(indexer.Identifier)
	if def == nil {
		return errors.New("could not find definition: %s", indexer.Identifier)
	}

	if !def.HasApi() {
		return errors.New("indexer (%s) does not support api", indexer.Identifier)
	}

	req.Identifier = def.Identifier

	if _, err = s.ApiService.TestConnection(ctx, req); err != nil {
		s.log.Error().Err(err).Msgf("error testing api for: %s", indexer.Identifier)
		return err
	}

	s.log.Info().Msgf("successful api test for: %s", indexer.Identifier)

	return nil
}

func (s *service) ToggleEnabled(ctx context.Context, indexerID int, enabled bool) error {
	indexer, err := s.FindByID(ctx, indexerID)
	if err != nil {
		return err
	}

	if err := s.repo.ToggleEnabled(ctx, int(indexer.ID), enabled); err != nil {
		s.log.Error().Err(err).Msg("could not update indexer enabled")
		return err
	}

	indexer.Enabled = enabled

	// update indexerInstances
	if err := s.updateIndexer(*indexer); err != nil {
		s.log.Error().Err(err).Msgf("failed to add indexer: %s", indexer.Name)
		return err
	}

	if isImplFeed(indexer.Implementation) {
		if !indexer.Enabled {
			s.stopFeed(indexer.Identifier)
		}
	}

	s.log.Debug().Msgf("indexer.toggle_enabled: update indexer '%d' to '%v'", indexerID, enabled)

	return nil
}

func isImplFeed(implementation string) bool {
	switch implementation {
	case "torznab", "newznab", "rss":
		return true
	default:
		return false
	}
}
