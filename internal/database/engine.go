package database

type EngineQuery struct {
	engine   string
	sqlite   string
	postgres string
}

func NewEngineQuery(engine string, sqlite, postgres string) *EngineQuery {
	return &EngineQuery{
		engine:   engine,
		sqlite:   sqlite,
		postgres: postgres,
	}
}

func (q *EngineQuery) Get() string {
	switch q.engine {
	case "sqlite":
		return q.sqlite
	case "postgres":
		return q.postgres
	}

	return ""
}
