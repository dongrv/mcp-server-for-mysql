package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultMaxOpenConns    = 10
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnMaxIdleTime = 5 * time.Minute
)

var (
	// ErrDuplicateSource is returned when more than one source has the same ID.
	ErrDuplicateSource = errors.New("duplicate source ID")
	// ErrUnknownSource is returned when a source ID is not registered.
	ErrUnknownSource = errors.New("unknown source")
)

// Registry owns all configured sources.
type Registry struct {
	sources  map[string]Source
	close    sync.Once
	closeErr error
}

// NewRegistry creates a registry from already opened sources.
// It closes all sources when their IDs are not unique.
func NewRegistry(sources []Source) (*Registry, error) {
	byID := make(map[string]Source, len(sources))
	for _, source := range sources {
		if _, exists := byID[source.ID()]; exists {
			closeSources(sources)
			return nil, fmt.Errorf("%w: %s", ErrDuplicateSource, source.ID())
		}
		byID[source.ID()] = source
	}
	return &Registry{sources: byID}, nil
}

// Source returns the configured source with id.
func (r *Registry) Source(id string) (Source, error) {
	source, ok := r.sources[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownSource, id)
	}
	return source, nil
}

// Sources returns every configured source ordered by ID. The returned slice is
// a snapshot and callers cannot mutate registry ownership through it.
func (r *Registry) Sources() []Source {
	if r == nil {
		return nil
	}
	ids := make([]string, 0, len(r.sources))
	for id := range r.sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	sources := make([]Source, 0, len(ids))
	for _, id := range ids {
		sources = append(sources, r.sources[id])
	}
	return sources
}

// Close closes every source exactly once and joins close failures.
func (r *Registry) Close() error {
	r.close.Do(func() {
		var errs []error
		for _, source := range r.sources {
			if err := source.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		r.closeErr = errors.Join(errs...)
	})
	return r.closeErr
}

type sourceFactory func(context.Context, config.SourceConfig) (Source, error)

// OpenRegistry opens and pings every configured source before returning it.
func OpenRegistry(ctx context.Context, cfg config.Config) (*Registry, error) {
	return openRegistry(ctx, cfg, openSource)
}

func openRegistry(ctx context.Context, cfg config.Config, factory sourceFactory) (*Registry, error) {
	sources := make([]Source, 0, len(cfg.Sources))
	for _, sourceConfig := range cfg.Sources {
		source, err := factory(ctx, sourceConfig)
		if err != nil {
			closeSources(sources)
			return nil, fmt.Errorf("open source %q: %w", sourceConfig.Name, err)
		}
		sources = append(sources, source)
	}
	registry, err := NewRegistry(sources)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func closeSources(sources []Source) {
	for _, source := range sources {
		_ = source.Close()
	}
}

type sqlSource struct {
	id       string
	engine   string
	profile  SourceProfile
	db       *sql.DB
	dialect  Dialect
	close    sync.Once
	closeErr error
}

func (s *sqlSource) ID() string               { return s.id }
func (s *sqlSource) Engine() string           { return s.engine }
func (s *sqlSource) Profile() SourceProfile   { return cloneSourceProfile(s.profile) }
func (s *sqlSource) DB() *sql.DB              { return s.db }
func (s *sqlSource) Dialect() Dialect         { return s.dialect }
func (s *sqlSource) Capabilities() Capability { return s.dialect.Capabilities() }
func (s *sqlSource) Close() error {
	s.close.Do(func() {
		s.closeErr = s.db.Close()
	})
	return s.closeErr
}

func openSource(ctx context.Context, cfg config.SourceConfig) (Source, error) {
	var (
		db      *sql.DB
		dialect Dialect
		err     error
	)
	switch cfg.Type {
	case "mysql":
		db, err = sql.Open("mysql", cfg.DSN)
		dialect = MySQLDialect{}
	case "clickhouse":
		db, err = openClickHouseDB(cfg.DSN)
		dialect = ClickHouseDialect{}
	default:
		return nil, errors.New("unsupported database engine")
	}
	if err != nil {
		return nil, errors.New("open database connection")
	}
	configurePool(db)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, errors.New("database connection check failed")
	}
	return &sqlSource{id: cfg.Name, engine: cfg.Type, profile: profileFromConfig(cfg), db: db, dialect: dialect}, nil
}

func profileFromConfig(cfg config.SourceConfig) SourceProfile {
	return cloneSourceProfile(SourceProfile{
		DisplayName: cfg.DisplayName,
		Description: cfg.Description,
		Aliases:     cfg.Aliases,
		Keywords:    cfg.Keywords,
	})
}

func cloneSourceProfile(profile SourceProfile) SourceProfile {
	if profile.Aliases != nil {
		profile.Aliases = append([]string{}, profile.Aliases...)
	}
	if profile.Keywords != nil {
		profile.Keywords = append([]string{}, profile.Keywords...)
	}
	return profile
}

func openClickHouseDB(dsn string) (*sql.DB, error) {
	options, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, errors.New("open database connection")
	}
	return clickhouse.OpenDB(options), nil
}

func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)
	db.SetConnMaxIdleTime(defaultConnMaxIdleTime)
}
