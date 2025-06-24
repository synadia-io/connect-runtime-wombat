package utils

import (
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

// NATSConnectionPool manages a pool of NATS connections for improved performance
type NATSConnectionPool struct {
	url         string
	opts        []nats.Option
	connections chan *nats.Conn
	maxConns    int
	mu          sync.RWMutex
	closed      bool
	logger      zerolog.Logger
}

// NATSPoolConfig holds configuration for the NATS connection pool
type NATSPoolConfig struct {
	URL         string
	MaxConns    int
	IdleTimeout time.Duration
	Options     []nats.Option
}

// NewNATSConnectionPool creates a new NATS connection pool
func NewNATSConnectionPool(config NATSPoolConfig) *NATSConnectionPool {
	if config.MaxConns <= 0 {
		config.MaxConns = 10 // Default pool size
	}

	pool := &NATSConnectionPool{
		url:         config.URL,
		opts:        config.Options,
		connections: make(chan *nats.Conn, config.MaxConns),
		maxConns:    config.MaxConns,
		logger:      InitLogger().With().Str("component", "nats_pool").Logger(),
	}

	pool.logger.Info().
		Str("url", config.URL).
		Int("max_connections", config.MaxConns).
		Msg("Created NATS connection pool")

	return pool
}

// Get retrieves a connection from the pool or creates a new one
func (p *NATSConnectionPool) Get(ctx context.Context) (*nats.Conn, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, nats.ErrConnectionClosed
	}
	p.mu.RUnlock()

	select {
	case conn := <-p.connections:
		if conn.IsConnected() {
			p.logger.Debug().Msg("Retrieved existing connection from pool")
			return conn, nil
		}
		// Connection is stale, close it and create a new one
		conn.Close()
		p.logger.Debug().Msg("Closed stale connection from pool")
	default:
	}

	// Create new connection (either no connection available or stale connection)
	conn, err := nats.Connect(p.url, p.opts...)
	if err != nil {
		p.logger.Error().Err(err).Msg("Failed to create new NATS connection")
		return nil, err
	}
	p.logger.Debug().Msg("Created new NATS connection")
	return conn, nil
}

// Put returns a connection to the pool
func (p *NATSConnectionPool) Put(conn *nats.Conn) {
	if conn == nil || !conn.IsConnected() {
		if conn != nil {
			conn.Close()
		}
		return
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		conn.Close()
		return
	}
	p.mu.RUnlock()

	select {
	case p.connections <- conn:
		p.logger.Debug().Msg("Returned connection to pool")
	default:
		// Pool is full, close the connection
		conn.Close()
		p.logger.Debug().Msg("Pool full, closed excess connection")
	}
}

// Close closes all connections in the pool
func (p *NATSConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.connections)

	// Close all connections in the pool
	for conn := range p.connections {
		if conn != nil {
			conn.Close()
		}
	}

	p.logger.Info().Msg("Closed NATS connection pool")
}

// Stats returns pool statistics
func (p *NATSConnectionPool) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"max_connections":       p.maxConns,
		"available_connections": len(p.connections),
		"closed":                p.closed,
		"url":                   p.url,
	}
}

// WithConnection executes a function with a pooled connection
func (p *NATSConnectionPool) WithConnection(ctx context.Context, fn func(*nats.Conn) error) error {
	conn, err := p.Get(ctx)
	if err != nil {
		return err
	}
	defer p.Put(conn)

	return fn(conn)
}
