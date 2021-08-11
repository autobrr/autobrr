package indexer

import (
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/rs/zerolog/log"

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
	repo                domain.IndexerRepo
	indexerDefinitions  map[string]domain.IndexerDefinition
	indexerInstances    map[string]domain.IndexerDefinition
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
		return nil, err
	}

	return i, nil
}

func (s *service) Update(indexer domain.Indexer) (*domain.Indexer, error) {
	i, err := s.repo.Update(indexer)
	if err != nil {
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
		in := s.getDefinitionByName(indexer.Identifier)
		if in == nil {
			// if no indexerDefinition found, continue
			continue
		}

		temp := domain.IndexerDefinition{
			ID:          indexer.ID,
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

				temp.SettingsMap[setting.Name] = v
			}

			temp.Settings = append(temp.Settings, setting)
		}

		res = append(res, &temp)
	}

	return res, nil
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

	indexers, err := s.GetAll()
	if err != nil {
		return err
	}

	for _, indexer := range indexers {
		if !indexer.Enabled {
			continue
		}

		s.indexerInstances[indexer.Identifier] = *indexer

		// map irc stuff to indexer.name
		if indexer.IRC != nil {
			server := indexer.IRC.Server

			for _, channel := range indexer.IRC.Channels {
				for _, announcer := range indexer.IRC.Announcers {
					val := fmt.Sprintf("%v:%v:%v", server, channel, announcer)
					s.mapIndexerIRCToName[val] = indexer.Identifier
				}
			}
		}
	}

	return nil
}

// LoadIndexerDefinitions load definitions from golang embed fs
func (s *service) LoadIndexerDefinitions() error {

	entries, err := fs.ReadDir(Definitions, "definitions")
	if err != nil {
		log.Fatal().Msgf("failed reading directory: %s", err)
	}

	if len(entries) == 0 {
		log.Fatal().Msgf("failed reading directory: %s", err)
		return err
	}

	for _, f := range entries {
		filePath := "definitions/" + f.Name()

		if strings.Contains(f.Name(), ".yaml") {
			log.Debug().Msgf("parsing: %v", filePath)

			var d domain.IndexerDefinition

			data, err := fs.ReadFile(Definitions, filePath)
			if err != nil {
				log.Debug().Err(err).Msgf("failed reading file: %v", filePath)
				return err
			}

			err = yaml.Unmarshal(data, &d)
			if err != nil {
				log.Error().Err(err).Msgf("failed unmarshal file: %v", filePath)
				return err
			}

			s.indexerDefinitions[d.Identifier] = d
		}
	}

	return nil
}

func (s *service) GetIndexerByAnnounce(name string) *domain.IndexerDefinition {

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
