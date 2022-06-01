package indexer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/scheduler"

	"github.com/gosimple/slug"
	"gopkg.in/yaml.v3"
)

type Service interface {
	Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error)
	Update(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error)
	Delete(ctx context.Context, id int) error
	FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error)
	List(ctx context.Context) ([]domain.Indexer, error)
	GetAll() ([]*domain.IndexerDefinition, error)
	GetTemplates() ([]*domain.IndexerDefinition, error)
	LoadIndexerDefinitions() error
	GetIndexersByIRCNetwork(server string) []*domain.IndexerDefinition
	GetTorznabIndexers() []domain.IndexerDefinition
	Start() error
}

type service struct {
	log        logger.Logger
	config     *domain.Config
	repo       domain.IndexerRepo
	apiService APIService
	scheduler  scheduler.Service

	// contains all raw indexer definitions
	indexerDefinitions map[string]*domain.IndexerDefinition

	// map server:channel:announce to indexer.Identifier
	mapIndexerIRCToName map[string]string

	lookupIRCServerDefinition map[string]map[string]*domain.IndexerDefinition

	torznabIndexers map[string]*domain.IndexerDefinition
}

func NewService(log logger.Logger, config *domain.Config, repo domain.IndexerRepo, apiService APIService, scheduler scheduler.Service) Service {
	return &service{
		log:                       log,
		config:                    config,
		repo:                      repo,
		apiService:                apiService,
		scheduler:                 scheduler,
		indexerDefinitions:        make(map[string]*domain.IndexerDefinition),
		mapIndexerIRCToName:       make(map[string]string),
		lookupIRCServerDefinition: make(map[string]map[string]*domain.IndexerDefinition),
		torznabIndexers:           make(map[string]*domain.IndexerDefinition),
	}
}

func (s *service) Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	identifier := indexer.Identifier
	if indexer.Identifier == "torznab" {
		// if the name already contains torznab remove it
		cleanName := strings.ReplaceAll(strings.ToLower(indexer.Name), "torznab", "")
		identifier = slug.Make(fmt.Sprintf("%v-%v", indexer.Identifier, cleanName))
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
		return nil, err
	}

	// add to indexerInstances
	err = s.addIndexer(*i)
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
		return err
	}

	// TODO remove handler if needed
	// remove from lookup tables

	return nil
}

func (s *service) FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error) {
	return s.repo.FindByFilterID(ctx, id)
}

func (s *service) List(ctx context.Context) ([]domain.Indexer, error) {
	return s.repo.List(ctx)
}

func (s *service) GetAll() ([]*domain.IndexerDefinition, error) {
	indexers, err := s.repo.List(context.Background())
	if err != nil {
		return nil, err
	}

	var res = make([]*domain.IndexerDefinition, 0)

	for _, indexer := range indexers {
		indexerDefinition, err := s.mapIndexer(indexer)
		if err != nil {
			continue
		}

		if indexerDefinition == nil {
			continue
		}

		res = append(res, indexerDefinition)
	}

	return res, nil
}

func (s *service) mapIndexer(indexer domain.Indexer) (*domain.IndexerDefinition, error) {

	var in *domain.IndexerDefinition
	if indexer.Implementation == "torznab" {
		in = s.getDefinitionByName("torznab")
		if in == nil {
			// if no indexerDefinition found, continue
			return nil, nil
		}
	} else {
		in = s.getDefinitionByName(indexer.Identifier)
		if in == nil {
			// if no indexerDefinition found, continue
			return nil, nil
		}
	}

	in.ID = int(indexer.ID)
	in.Name = indexer.Name
	in.Identifier = indexer.Identifier
	in.Implementation = indexer.Implementation
	in.Enabled = indexer.Enabled
	in.SettingsMap = make(map[string]string)

	if in.Implementation == "" {
		in.Implementation = "irc"
	}

	// map settings
	// add value to settings objects
	for i, setting := range in.Settings {
		if v, ok := indexer.Settings[setting.Name]; ok {
			setting.Value = v

			in.SettingsMap[setting.Name] = v
		}

		in.Settings[i] = setting
	}

	return in, nil
}

func (s *service) GetTemplates() ([]*domain.IndexerDefinition, error) {

	definitions := s.indexerDefinitions

	ret := make([]*domain.IndexerDefinition, 0)
	for _, definition := range definitions {
		ret = append(ret, definition)
	}

	return ret, nil
}

func (s *service) Start() error {
	// load all indexer definitions
	err := s.LoadIndexerDefinitions()
	if err != nil {
		return err
	}

	if s.config.CustomDefinitions != "" {
		// load custom indexer definitions
		err = s.LoadCustomIndexerDefinitions()
		if err != nil {
			return fmt.Errorf("could not load custom indexer definitions: %w", err)
		}
	}

	// load the indexers' setup by the user
	indexerDefinitions, err := s.GetAll()
	if err != nil {
		return err
	}

	for _, indexer := range indexerDefinitions {
		if indexer.IRC != nil {
			s.mapIRCIndexerLookup(indexer.Identifier, indexer)

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

	delete(s.indexerDefinitions, indexer.Identifier)

	// TODO delete from mapIndexerIRCToName

	return nil
}

func (s *service) addIndexer(indexer domain.Indexer) error {

	// TODO only add if not already there?? Overwrite?

	indexerDefinition, err := s.mapIndexer(indexer)
	if err != nil {
		return err
	}

	if indexerDefinition == nil {
		return errors.New("addindexer: could not find definition")
	}

	// TODO only add enabled?
	//if !indexer.Enabled {
	//	continue
	//}

	if indexerDefinition.IRC != nil {
		s.mapIRCIndexerLookup(indexer.Identifier, indexerDefinition)

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

	return nil
}

func (s *service) mapIRCIndexerLookup(indexerIdentifier string, indexerDefinition *domain.IndexerDefinition) {
	// map irc stuff to indexer.name
	// map[irc.network.test:channel:announcer1] = indexer1
	// map[irc.network.test:channel:announcer2] = indexer2
	if indexerDefinition.IRC != nil {
		server := indexerDefinition.IRC.Server
		channels := indexerDefinition.IRC.Channels
		announcers := indexerDefinition.IRC.Announcers

		for _, channel := range channels {
			for _, announcer := range announcers {
				// format to server:channel:announcer
				val := fmt.Sprintf("%v:%v:%v", server, channel, announcer)
				val = strings.ToLower(val)

				s.mapIndexerIRCToName[val] = indexerIdentifier
			}
		}
	}
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
		s.log.Fatal().Stack().Msgf("failed reading directory: %s", err)
	}

	if len(entries) == 0 {
		s.log.Fatal().Stack().Msgf("failed reading directory: %s", err)
		return err
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
			return err
		}

		err = yaml.Unmarshal(data, &d)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("failed unmarshal file: %v", file)
			return err
		}

		if d.Implementation == "" {
			d.Implementation = "irc"
		}

		s.indexerDefinitions[d.Identifier] = d
	}

	s.log.Debug().Msgf("Loaded %d indexer definitions", len(s.indexerDefinitions))

	return nil
}

// LoadCustomIndexerDefinitions load definitions from custom path
func (s *service) LoadCustomIndexerDefinitions() error {
	if s.config.CustomDefinitions == "" {
		return nil
	}

	outputDirRead, _ := os.Open(s.config.CustomDefinitions)

	//entries, err := fs.ReadDir(Definitions, "definitions")
	entries, err := outputDirRead.ReadDir(0)
	if err != nil {
		s.log.Fatal().Stack().Msgf("failed reading directory: %s", err)
	}

	if len(entries) == 0 {
		s.log.Fatal().Stack().Msgf("failed reading directory: %s", err)
		return err
	}

	customCount := 0

	for _, f := range entries {
		fileExtension := filepath.Ext(f.Name())
		if fileExtension != ".yaml" {
			continue
		}

		file := filepath.Join(s.config.CustomDefinitions, f.Name())

		s.log.Trace().Msgf("parsing custom: %v", file)

		var d *domain.IndexerDefinition

		//data, err := fs.ReadFile(Definitions, filePath)
		data, err := os.ReadFile(file)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("failed reading file: %v", file)
			return err
		}

		err = yaml.Unmarshal(data, &d)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("failed unmarshal file: %v", file)
			return err
		}

		if d.Implementation == "" {
			d.Implementation = "irc"
		}

		s.indexerDefinitions[d.Identifier] = d

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

	if v, ok := s.indexerDefinitions[name]; ok {
		return v
	}

	return nil
}

func (s *service) getDefinitionForAnnounce(name string) *domain.IndexerDefinition {

	// map[network:channel:announcer] = indexer01

	if v, ok := s.indexerDefinitions[name]; ok {
		return v
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
