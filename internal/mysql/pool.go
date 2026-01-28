// Package mysql provides MySQL database connection management.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	_ "github.com/go-sql-driver/mysql"
)

// Pool manages a pool of MySQL database connections.
type Pool struct {
	db   *sql.DB
	mu   sync.RWMutex
	cfg  *config.MySQLConfig
	stop chan struct{}
}

// NewPool creates a new MySQL connection pool.
func NewPool(cfg *config.MySQLConfig) (*Pool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pool := &Pool{
		db:   db,
		cfg:  cfg,
		stop: make(chan struct{}),
	}

	return pool, nil
}

// DB returns the underlying sql.DB instance.
// This method is safe for concurrent use.
func (p *Pool) DB() *sql.DB {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.db
}

// Close closes the connection pool and releases all resources.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db == nil {
		return nil
	}

	close(p.stop)
	err := p.db.Close()
	p.db = nil
	return err
}

// Stats returns database connection pool statistics.
func (p *Pool) Stats() sql.DBStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.db != nil {
		return p.db.Stats()
	}
	return sql.DBStats{}
}

// Config returns a copy of the pool's configuration.
func (p *Pool) Config() config.MySQLConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return *p.cfg
}

// HealthCheck performs a health check on the database connection.
func (p *Pool) HealthCheck(ctx context.Context) error {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database pool is closed")
	}

	return db.PingContext(ctx)
}

// BeginTx starts a new transaction with the given context and options.
func (p *Pool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database pool is closed")
	}

	return db.BeginTx(ctx, opts)
}

// ExecContext executes a query without returning any rows.
func (p *Pool) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database pool is closed")
	}

	return db.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows.
func (p *Pool) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database pool is closed")
	}

	return db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row.
func (p *Pool) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		// Return a Row that will error when scanned
		return &sql.Row{}
	}

	return db.QueryRowContext(ctx, query, args...)
}

// WithTransaction executes a function within a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (p *Pool) WithTransaction(ctx context.Context, opts *sql.TxOptions, fn func(*sql.Tx) error) error {
	tx, err := p.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
