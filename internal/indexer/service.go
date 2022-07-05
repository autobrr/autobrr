package indexer

import (
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

	"github.com/gosimple/slug"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

type Service interface {
	Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error)
	Update(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error)
	Delete(ctx context.Context, id int) error
	FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error)
	List(ctx context.Context) ([]domain.Indexer, error)
	GetAll() ([]*domain.IndexerDefinition, error)
	GetTemplates() ([]domain.IndexerDefinition, error)
	LoadIndexerDefinitions() error
	GetIndexersByIRCNetwork(server string) []*domain.IndexerDefinition
	GetTorznabIndexers() []domain.IndexerDefinition
	Start() error
}

type service struct {
	log        zerolog.Logger
	config     *domain.Config
	repo       domain.IndexerRepo
	apiService APIService
	scheduler  scheduler.Service

	// contains all raw indexer definitions
	definitions map[string]domain.IndexerDefinition
	// definition with indexer data
	mappedDefinitions map[string]*domain.IndexerDefinition
	// map server:channel:announce to indexer.Identifier
	lookupIRCServerDefinition map[string]map[string]*domain.IndexerDefinition
	// torznab indexers
	torznabIndexers map[string]*domain.IndexerDefinition
}

func NewService(log logger.Logger, config *domain.Config, repo domain.IndexerRepo, apiService APIService, scheduler scheduler.Service) Service {
	return &service{
		log:                       log.With().Str("module", "indexer").Logger(),
		config:                    config,
		repo:                      repo,
		apiService:                apiService,
		scheduler:                 scheduler,
		lookupIRCServerDefinition: make(map[string]map[string]*domain.IndexerDefinition),
		torznabIndexers:           make(map[string]*domain.IndexerDefinition),
		definitions:               make(map[string]domain.IndexerDefinition),
		mappedDefinitions:         make(map[string]*domain.IndexerDefinition),
	}
}

func (s *service) Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	identifier := indexer.Identifier
	//if indexer.Identifier == "torznab" {
	if indexer.Implementation == "torznab" {
		// if the name already contains torznab remove it
		cleanName := strings.ReplaceAll(strings.ToLower(indexer.Name), "torznab", "")
		identifier = slug.Make(fmt.Sprintf("%v-%v", indexer.Implementation, cleanName)) // torznab-name
	}

	indexer.Identifier = identifier

	i, err := s.repo.Store(ctx, indexer)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("failed to store indexer: %v", indexer.Name)
		return nil, err
	}

	// add to indexerInstances
	err = s.addIndexer(*i)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("failed to add indexer: %v", indexer.Name)
		return nil, err
	}

	return i, nil
}

func (s *service) Update(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	i, err := s.repo.Update(ctx, indexer)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not update indexer: %+v", indexer)
		return nil, err
	}

	// add to indexerInstances
	err = s.updateIndexer(*i)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("failed to add indexer: %v", indexer.Name)
		return nil, err
	}

	if indexer.Implementation == "torznab" {
		if !indexer.Enabled {
			s.stopFeed(indexer.Identifier)
		}
	}

	return i, nil
}

func (s *service) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.log.Error().Err(err).Msgf("could not delete indexer by id: %v", id)
		return err
	}

	// TODO remove handler if needed
	// remove from lookup tables

	return nil
}

func (s *service) FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error) {
	indexers, err := s.repo.FindByFilterID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find indexers by filter id: %v", id)
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
	if indexer.Implementation == "torznab" {
		definitionName = "torznab"
	}

	d := s.getDefinitionByName(definitionName)
	if d == nil {
		// if no indexerDefinition found, continue
		return nil, nil
	}

	d.ID = int(indexer.ID)
	d.Name = indexer.Name
	d.Identifier = indexer.Identifier
	d.Implementation = indexer.Implementation
	d.Enabled = indexer.Enabled

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
	d.Implementation = indexer.Implementation
	d.Enabled = indexer.Enabled

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
	err := s.LoadIndexerDefinitions()
	if err != nil {
		s.log.Error().Err(err).Msg("could not load indexer definitions")
		return err
	}

	if s.config.CustomDefinitions != "" {
		// load custom indexer definitions
		err = s.LoadCustomIndexerDefinitions()
		if err != nil {
			return errors.Wrap(err, "could not load custom indexer definitions")
		}
	}

	// load the indexers' setup by the user
	indexerDefinitions, err := s.mapIndexers()
	if err != nil {
		return err
	}

	for _, indexer := range indexerDefinitions {
		if indexer.IRC != nil {
			// add to irc server lookup table
			s.mapIRCServerDefinitionLookup(indexer.IRC.Server, indexer)

			// check if it has api and add to api service
			if indexer.Enabled && indexer.HasApi() {
				if err := s.apiService.AddClient(indexer.Identifier, indexer.SettingsMap); err != nil {
					s.log.Error().Stack().Err(err).Msgf("indexer.start: could not init api client for: '%v'", indexer.Identifier)
				}
			}
		}

		// handle Torznab
		if indexer.Implementation == "torznab" {
			s.torznabIndexers[indexer.Identifier] = indexer
		}
	}

	s.log.Info().Msgf("Loaded %d indexers", len(indexerDefinitions))

	return nil
}

func (s *service) removeIndexer(indexer domain.Indexer) error {
	delete(s.definitions, indexer.Identifier)

	return nil
}

func (s *service) addIndexer(indexer domain.Indexer) error {
	indexerDefinition, err := s.mapIndexer(indexer)
	if err != nil {
		return err
	}

	if indexerDefinition == nil {
		return errors.New("addindexer: could not find definition")
	}

	if indexerDefinition.IRC != nil {
		// add to irc server lookup table
		s.mapIRCServerDefinitionLookup(indexerDefinition.IRC.Server, indexerDefinition)

		// check if it has api and add to api service
		if indexerDefinition.Enabled && indexerDefinition.HasApi() {
			if err := s.apiService.AddClient(indexerDefinition.Identifier, indexerDefinition.SettingsMap); err != nil {
				s.log.Error().Stack().Err(err).Msgf("indexer.start: could not init api client for: '%v'", indexer.Identifier)
			}
		}
	}

	// handle Torznab
	if indexerDefinition.Implementation == "torznab" {
		s.torznabIndexers[indexer.Identifier] = indexerDefinition
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

	if indexerDefinition.IRC != nil {
		// add to irc server lookup table
		s.mapIRCServerDefinitionLookup(indexerDefinition.IRC.Server, indexerDefinition)

		// check if it has api and add to api service
		if indexerDefinition.Enabled && indexerDefinition.HasApi() {
			if err := s.apiService.AddClient(indexerDefinition.Identifier, indexerDefinition.SettingsMap); err != nil {
				s.log.Error().Stack().Err(err).Msgf("indexer.start: could not init api client for: '%v'", indexer.Identifier)
			}
		}
	}

	// handle Torznab
	if indexerDefinition.Implementation == "torznab" {
		s.torznabIndexers[indexer.Identifier] = indexerDefinition
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

		s.log.Trace().Msgf("parsing: %v", file)

		var d *domain.IndexerDefinition

		data, err := fs.ReadFile(Definitions, file)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("failed reading file: %v", file)
			return errors.Wrap(err, "could not read file: %v", file)
		}

		err = yaml.Unmarshal(data, &d)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("failed unmarshal file: %v", file)
			return errors.Wrap(err, "could not unmarshal file: %v", file)
		}

		if d.Implementation == "" {
			d.Implementation = "irc"
		}

		s.definitions[d.Identifier] = *d
	}

	s.log.Debug().Msgf("Loaded %d indexer definitions", len(s.definitions))

	return nil
}

// LoadCustomIndexerDefinitions load definitions from custom path
func (s *service) LoadCustomIndexerDefinitions() error {
	if s.config.CustomDefinitions == "" {
		return nil
	}

	outputDirRead, err := os.Open(s.config.CustomDefinitions)
	if err != nil {
		s.log.Warn().Stack().Msgf("failed opening custom definitions directory %q: %s", s.config.CustomDefinitions, err)
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
		fileExtension := filepath.Ext(f.Name())
		if fileExtension != ".yaml" && fileExtension != ".yml" {
			s.log.Warn().Stack().Msgf("skipping unknown extension definition file: %s", f.Name())
			continue
		}

		file := filepath.Join(s.config.CustomDefinitions, f.Name())

		s.log.Trace().Msgf("parsing custom: %v", file)

		//data, err := fs.ReadFile(Definitions, filePath)
		data, err := os.ReadFile(file)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("failed reading file: %v", file)
			return errors.Wrap(err, "could not read file: %v", file)
		}

		var d *domain.IndexerDefinition
		if err = yaml.Unmarshal(data, &d); err != nil {
			s.log.Error().Stack().Err(err).Msgf("failed unmarshal file: %v", file)
			return errors.Wrap(err, "could not unmarshal file: %v", file)
		}

		if d == nil {
			s.log.Warn().Stack().Err(err).Msgf("skipping empty file: %v", file)
			continue
		}

		if d.Implementation == "" {
			d.Implementation = "irc"
		}

		s.definitions[d.Identifier] = *d

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

func (s *service) getDefinitionByName(name string) *domain.IndexerDefinition {
	if v, ok := s.definitions[name]; ok {
		return &v
	}

	return nil
}

func (s *service) stopFeed(indexer string) {
	// verify indexer is torznab indexer
	_, ok := s.torznabIndexers[indexer]
	if !ok {
		return
	}

	if err := s.scheduler.RemoveJobByIdentifier(indexer); err != nil {
		return
	}
}
