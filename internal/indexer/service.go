package indexer

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"

	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	Store(indexer domain.Indexer) (*domain.Indexer, error)
	Update(indexer domain.Indexer) (*domain.Indexer, error)
	Delete(ctx context.Context, id int) error
	FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error)
	List() ([]domain.Indexer, error)
	GetAll() ([]*domain.IndexerDefinition, error)
	GetTemplates() ([]domain.IndexerDefinition, error)
	LoadIndexerDefinitions() error
	GetIndexersByIRCNetwork(server string) []domain.IndexerDefinition
	Start() error
}

type service struct {
	config     domain.Config
	repo       domain.IndexerRepo
	apiService APIService

	// contains all raw indexer definitions
	indexerDefinitions map[string]domain.IndexerDefinition

	// map server:channel:announce to indexer.Identifier
	mapIndexerIRCToName map[string]string

	lookupIRCServerDefinition map[string]map[string]domain.IndexerDefinition
}

func NewService(config domain.Config, repo domain.IndexerRepo, apiService APIService) Service {
	return &service{
		config:                    config,
		repo:                      repo,
		apiService:                apiService,
		indexerDefinitions:        make(map[string]domain.IndexerDefinition),
		mapIndexerIRCToName:       make(map[string]string),
		lookupIRCServerDefinition: make(map[string]map[string]domain.IndexerDefinition),
	}
}

func (s *service) Store(indexer domain.Indexer) (*domain.Indexer, error) {
	i, err := s.repo.Store(indexer)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("failed to store indexer: %v", indexer.Name)
		return nil, err
	}

	// add to indexerInstances
	err = s.addIndexer(*i)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("failed to add indexer: %v", indexer.Name)
		return nil, err
	}

	return i, nil
}

func (s *service) Update(indexer domain.Indexer) (*domain.Indexer, error) {
	i, err := s.repo.Update(indexer)
	if err != nil {
		return nil, err
	}

	// add to indexerInstances
	err = s.addIndexer(*i)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("failed to add indexer: %v", indexer.Name)
		return nil, err
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
	filters, err := s.repo.FindByFilterID(ctx, id)
	if err != nil {
		return nil, err
	}

	return filters, nil
}

func (s *service) List() ([]domain.Indexer, error) {
	i, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (s *service) GetAll() ([]*domain.IndexerDefinition, error) {
	indexers, err := s.repo.List()
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

	in := s.getDefinitionByName(indexer.Identifier)
	if in == nil {
		// if no indexerDefinition found, continue
		return nil, nil
	}

	indexerDefinition := domain.IndexerDefinition{
		ID:          int(indexer.ID),
		Name:        in.Name,
		Identifier:  in.Identifier,
		Enabled:     indexer.Enabled,
		Description: in.Description,
		Language:    in.Language,
		Privacy:     in.Privacy,
		Protocol:    in.Protocol,
		URLS:        in.URLS,
		Supports:    in.Supports,
		Settings:    nil,
		SettingsMap: make(map[string]string),
		IRC:         in.IRC,
		Parse:       in.Parse,
	}

	// map settings
	// add value to settings objects
	for _, setting := range in.Settings {
		if v, ok := indexer.Settings[setting.Name]; ok {
			setting.Value = v

			indexerDefinition.SettingsMap[setting.Name] = v
		}

		indexerDefinition.Settings = append(indexerDefinition.Settings, setting)
	}

	return &indexerDefinition, nil
}

func (s *service) GetTemplates() ([]domain.IndexerDefinition, error) {

	definitions := s.indexerDefinitions

	var ret []domain.IndexerDefinition
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
		s.mapIRCIndexerLookup(indexer.Identifier, *indexer)

		// add to irc server lookup table
		s.mapIRCServerDefinitionLookup(indexer.IRC.Server, *indexer)

		// check if it has api and add to api service
		if indexer.Enabled && indexer.HasApi() {
			if err := s.apiService.AddClient(indexer.Identifier, indexer.SettingsMap); err != nil {
				log.Error().Stack().Err(err).Msgf("indexer.start: could not init api client for: '%v'", indexer.Identifier)
			}
		}
	}

	log.Info().Msgf("Loaded %d indexers", len(indexerDefinitions))

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

	// TODO only add enabled?
	//if !indexer.Enabled {
	//	continue
	//}

	s.mapIRCIndexerLookup(indexer.Identifier, *indexerDefinition)

	// add to irc server lookup table
	s.mapIRCServerDefinitionLookup(indexerDefinition.IRC.Server, *indexerDefinition)

	// check if it has api and add to api service
	if indexerDefinition.Enabled && indexerDefinition.HasApi() {
		if err := s.apiService.AddClient(indexerDefinition.Identifier, indexerDefinition.SettingsMap); err != nil {
			log.Error().Stack().Err(err).Msgf("indexer.start: could not init api client for: '%v'", indexer.Identifier)
		}
	}

	return nil
}

func (s *service) mapIRCIndexerLookup(indexerIdentifier string, indexerDefinition domain.IndexerDefinition) {
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
func (s *service) mapIRCServerDefinitionLookup(ircServer string, indexerDefinition domain.IndexerDefinition) {
	if indexerDefinition.IRC != nil {
		// check if already exists, if ok add it to existing, otherwise create new
		_, exists := s.lookupIRCServerDefinition[ircServer]
		if !exists {
			s.lookupIRCServerDefinition[ircServer] = map[string]domain.IndexerDefinition{}
		}

		s.lookupIRCServerDefinition[ircServer][indexerDefinition.Identifier] = indexerDefinition
	}
}

// LoadIndexerDefinitions load definitions from golang embed fs
func (s *service) LoadIndexerDefinitions() error {

	entries, err := fs.ReadDir(Definitions, "definitions")
	if err != nil {
		log.Fatal().Stack().Msgf("failed reading directory: %s", err)
	}

	if len(entries) == 0 {
		log.Fatal().Stack().Msgf("failed reading directory: %s", err)
		return err
	}

	for _, f := range entries {
		filePath := "definitions/" + f.Name()

		if strings.Contains(f.Name(), ".yaml") {
			log.Trace().Msgf("parsing: %v", filePath)

			var d domain.IndexerDefinition

			data, err := fs.ReadFile(Definitions, filePath)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("failed reading file: %v", filePath)
				return err
			}

			err = yaml.Unmarshal(data, &d)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("failed unmarshal file: %v", filePath)
				return err
			}

			s.indexerDefinitions[d.Identifier] = d
		}
	}

	log.Debug().Msgf("Loaded %d indexer definitions", len(s.indexerDefinitions))

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
		log.Fatal().Stack().Msgf("failed reading directory: %s", err)
	}

	if len(entries) == 0 {
		log.Fatal().Stack().Msgf("failed reading directory: %s", err)
		return err
	}

	customCount := 0

	for _, f := range entries {
		file := filepath.Join(s.config.CustomDefinitions, f.Name())

		if strings.Contains(f.Name(), ".yaml") {
			log.Trace().Msgf("parsing custom: %v", file)

			var d domain.IndexerDefinition

			//data, err := fs.ReadFile(Definitions, filePath)
			data, err := os.ReadFile(file)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("failed reading file: %v", file)
				return err
			}

			err = yaml.Unmarshal(data, &d)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("failed unmarshal file: %v", file)
				return err
			}

			s.indexerDefinitions[d.Identifier] = d

			customCount++
		}
	}

	log.Debug().Msgf("Loaded %d custom indexer definitions", customCount)

	return nil
}

func (s *service) GetIndexersByIRCNetwork(server string) []domain.IndexerDefinition {
	server = strings.ToLower(server)

	indexerDefinitions := make([]domain.IndexerDefinition, 0)

	// get indexer definitions matching irc network from lookup table
	if srv, idOk := s.lookupIRCServerDefinition[server]; idOk {
		for _, definition := range srv {
			indexerDefinitions = append(indexerDefinitions, definition)
		}
	}

	return indexerDefinitions
}

func (s *service) getDefinitionByName(name string) *domain.IndexerDefinition {

	if v, ok := s.indexerDefinitions[name]; ok {
		return &v
	}

	return nil
}

func (s *service) getDefinitionForAnnounce(name string) *domain.IndexerDefinition {

	// map[network:channel:announcer] = indexer01

	if v, ok := s.indexerDefinitions[name]; ok {
		return &v
	}

	return nil
}
