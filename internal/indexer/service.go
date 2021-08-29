package indexer

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"

	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	Store(indexer domain.Indexer) (*domain.Indexer, error)
	Update(indexer domain.Indexer) (*domain.Indexer, error)
	Delete(id int) error
	FindByFilterID(id int) ([]domain.Indexer, error)
	List() ([]domain.Indexer, error)
	GetAll() ([]*domain.IndexerDefinition, error)
	GetTemplates() ([]domain.IndexerDefinition, error)
	LoadIndexerDefinitions() error
	GetIndexerByAnnounce(name string) *domain.IndexerDefinition
	Start() error
}

type service struct {
	repo domain.IndexerRepo

	// contains all raw indexer definitions
	indexerDefinitions map[string]domain.IndexerDefinition

	// contains indexers with data set
	indexerInstances map[string]domain.IndexerDefinition

	// map server:channel:announce to indexer.Identifier
	mapIndexerIRCToName map[string]string
}

func NewService(repo domain.IndexerRepo) Service {
	return &service{
		repo:                repo,
		indexerDefinitions:  make(map[string]domain.IndexerDefinition),
		indexerInstances:    make(map[string]domain.IndexerDefinition),
		mapIndexerIRCToName: make(map[string]string),
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

func (s *service) Delete(id int) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}

	return nil
}

func (s *service) FindByFilterID(id int) ([]domain.Indexer, error) {
	filters, err := s.repo.FindByFilterID(id)
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
	err := s.LoadIndexerDefinitions()
	if err != nil {
		return err
	}

	indexerDefinitions, err := s.GetAll()
	if err != nil {
		return err
	}

	for _, indexerDefinition := range indexerDefinitions {
		s.indexerInstances[indexerDefinition.Identifier] = *indexerDefinition

		s.mapIRCIndexerLookup(indexerDefinition.Identifier, *indexerDefinition)
	}

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

	s.indexerInstances[indexerDefinition.Identifier] = *indexerDefinition

	s.mapIRCIndexerLookup(indexer.Identifier, *indexerDefinition)

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

	log.Info().Msgf("Loaded %d indexer definitions", len(s.indexerDefinitions))

	return nil
}

func (s *service) GetIndexerByAnnounce(name string) *domain.IndexerDefinition {
	name = strings.ToLower(name)

	if identifier, idOk := s.mapIndexerIRCToName[name]; idOk {
		if indexer, ok := s.indexerInstances[identifier]; ok {
			return &indexer
		}
	}

	return nil
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
