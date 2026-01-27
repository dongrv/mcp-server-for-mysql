// Package mysql provides MySQL database connection management.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// TxManager manages database transactions with unique IDs.
type TxManager struct {
	mu           sync.RWMutex
	transactions map[string]*sql.Tx
	pool         *Pool
}

// NewTxManager creates a new transaction manager.
func NewTxManager(pool *Pool) *TxManager {
	return &TxManager{
		transactions: make(map[string]*sql.Tx),
		pool:         pool,
	}
}

// Begin starts a new transaction and returns a transaction ID.
func (tm *TxManager) Begin(ctx context.Context) (string, error) {
	tx, err := tm.pool.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	txID := generateTxID()
	tm.mu.Lock()
	tm.transactions[txID] = tx
	tm.mu.Unlock()

	return txID, nil
}

// Commit commits the transaction with the given ID.
func (tm *TxManager) Commit(ctx context.Context, txID string) error {
	tm.mu.Lock()
	tx, exists := tm.transactions[txID]
	if !exists {
		tm.mu.Unlock()
		return fmt.Errorf("transaction not found: %s", txID)
	}
	delete(tm.transactions, txID)
	tm.mu.Unlock()

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback rolls back the transaction with the given ID.
func (tm *TxManager) Rollback(ctx context.Context, txID string) error {
	tm.mu.Lock()
	tx, exists := tm.transactions[txID]
	if !exists {
		tm.mu.Unlock()
		return fmt.Errorf("transaction not found: %s", txID)
	}
	delete(tm.transactions, txID)
	tm.mu.Unlock()

	if err := tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// Get returns the transaction with the given ID.
func (tm *TxManager) Get(txID string) (*sql.Tx, bool) {
	tm.mu.RLock()
	tx, exists := tm.transactions[txID]
	tm.mu.RUnlock()
	return tx, exists
}

// Cleanup removes all transactions that have been idle for too long.
func (tm *TxManager) Cleanup(maxIdle time.Duration) {
	// This is a simplified implementation.
	// In production, you might want to track transaction creation time
	// and clean up old transactions.
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// For now, we just clear all transactions
	// This should be called when the server shuts down
	for txID, tx := range tm.transactions {
		tx.Rollback()
		delete(tm.transactions, txID)
	}
}

// Count returns the number of active transactions.
func (tm *TxManager) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.transactions)
}

// generateTxID generates a unique transaction ID.
func generateTxID() string {
	return fmt.Sprintf("tx_%d", time.Now().UnixNano())
}
