// Package storage provides optimized database connection pooling
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds database connection pool configuration
type PoolConfig struct {
	// Maximum number of connections in the pool
	MaxConns int32

	// Minimum number of idle connections
	MinConns int32

	// Maximum lifetime of a connection
	MaxConnLifetime time.Duration

	// Maximum idle time of a connection
	MaxConnIdleTime time.Duration

	// Health check period
	HealthCheckPeriod time.Duration

	// Connection timeout
	ConnectTimeout time.Duration
}

// DefaultPoolConfig returns optimized default pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:          25,                // Handle 25 concurrent requests
		MinConns:          5,                 // Keep 5 connections warm
		MaxConnLifetime:   1 * time.Hour,    // Rotate connections hourly
		MaxConnIdleTime:   15 * time.Minute, // Close idle connections after 15 min
		HealthCheckPeriod: 1 * time.Minute,  // Check health every minute
		ConnectTimeout:    10 * time.Second, // Timeout connection attempts
	}
}

// EnterprisePoolConfig returns configuration for high-load enterprise deployments
func EnterprisePoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:          100,               // Handle 100 concurrent requests
		MinConns:          10,                // Keep 10 connections warm
		MaxConnLifetime:   30 * time.Minute, // Rotate more frequently
		MaxConnIdleTime:   5 * time.Minute,  // Close idle connections quickly
		HealthCheckPeriod: 30 * time.Second, // Check health more frequently
		ConnectTimeout:    5 * time.Second,  // Faster timeout for high load
	}
}

// CreateOptimizedPool creates a pgxpool with optimized settings
func CreateOptimizedPool(ctx context.Context, connString string, config PoolConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Apply optimized pool settings
	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinConns = config.MinConns
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod
	poolConfig.ConnConfig.ConnectTimeout = config.ConnectTimeout

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// PoolStats holds connection pool statistics
type PoolStats struct {
	AcquireCount         int64
	AcquireDuration      time.Duration
	AcquiredConns        int32
	CanceledAcquireCount int64
	ConstructingConns    int32
	EmptyAcquireCount    int64
	IdleConns            int32
	MaxConns             int32
	TotalConns           int32
}

// GetPoolStats returns current pool statistics
func GetPoolStats(pool *pgxpool.Pool) PoolStats {
	stats := pool.Stat()
	return PoolStats{
		AcquireCount:         stats.AcquireCount(),
		AcquireDuration:      stats.AcquireDuration(),
		AcquiredConns:        stats.AcquiredConns(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
		ConstructingConns:    stats.ConstructingConns(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		IdleConns:            stats.IdleConns(),
		MaxConns:             stats.MaxConns(),
		TotalConns:           stats.TotalConns(),
	}
}

// HealthCheck performs a comprehensive health check on the pool
func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	// Check if we can ping the database
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Check pool statistics
	stats := pool.Stat()
	
	// Warn if we're consistently hitting max connections
	if stats.AcquiredConns() >= stats.MaxConns() {
		return fmt.Errorf("pool exhausted: %d/%d connections in use", stats.AcquiredConns(), stats.MaxConns())
	}

	// Warn if we have too many idle connections
	if stats.IdleConns() > stats.MaxConns()/2 && stats.IdleConns() > 10 {
		// This is just a warning, not an error
		fmt.Printf("Warning: High number of idle connections: %d/%d\n", stats.IdleConns(), stats.MaxConns())
	}

	return nil
}

// QueryOptimizer provides helpers for query optimization
type QueryOptimizer struct{}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer() *QueryOptimizer {
	return &QueryOptimizer{}
}

// PreparedStatement helpers for common queries
const (
	// User queries
	GetUserByIDQuery = `
		SELECT id, username, email, created_at, updated_at 
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL
	`

	// Workspace queries with index hints
	GetWorkspaceByIDQuery = `
		SELECT id, name, user_id, created_at, updated_at
		FROM workspaces
		WHERE id = $1 AND deleted_at IS NULL
	`

	// MCP Server queries optimized for common patterns
	ListMCPServersByUserQuery = `
		SELECT id, name, manifest, status, created_at
		FROM mcp_servers
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	// Quota queries
	GetQuotaUsageQuery = `
		SELECT resource_type, current_usage, quota_limit, reset_at
		FROM quota_usage
		WHERE user_id = $1
	`

	// Cost queries with aggregation
	GetCostReportQuery = `
		SELECT 
			DATE(timestamp) as date,
			SUM(cost_usd) as total_cost,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(completion_tokens) as completion_tokens
		FROM llm_usage_events
		WHERE user_id = $1 
		  AND timestamp >= $2 
		  AND timestamp <= $3
		GROUP BY DATE(timestamp)
		ORDER BY date DESC
	`
)

// IndexDefinitions for performance optimization
var RecommendedIndexes = []string{
	// User indexes
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;",
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL;",

	// Workspace indexes
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workspaces_user_id ON workspaces(user_id) WHERE deleted_at IS NULL;",
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workspaces_created_at ON workspaces(created_at DESC);",

	// MCP Server indexes
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mcp_servers_user_id ON mcp_servers(user_id) WHERE deleted_at IS NULL;",
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mcp_servers_status ON mcp_servers(status) WHERE deleted_at IS NULL;",
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mcp_servers_created_at ON mcp_servers(created_at DESC);",

	// Thread indexes
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_threads_workspace_id ON threads(workspace_id) WHERE deleted_at IS NULL;",
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_threads_user_id ON threads(user_id) WHERE deleted_at IS NULL;",

	// Quota indexes
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_quota_usage_user_id ON quota_usage(user_id);",
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_quota_usage_reset_at ON quota_usage(reset_at);",

	// Cost tracking indexes
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_llm_usage_user_id_timestamp ON llm_usage_events(user_id, timestamp DESC);",
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_llm_usage_workspace_id ON llm_usage_events(workspace_id);",
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_llm_usage_model_provider ON llm_usage_events(model_provider, model_name);",

	// Audit log indexes
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_user_id_timestamp ON audit_logs(user_id, timestamp DESC);",
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type, resource_id);",

	// Composite indexes for common query patterns
	"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mcp_servers_user_status ON mcp_servers(user_id, status) WHERE deleted_at IS NULL;",
}

// CreateRecommendedIndexes creates all recommended indexes
func CreateRecommendedIndexes(ctx context.Context, pool *pgxpool.Pool) error {
	for _, indexSQL := range RecommendedIndexes {
		if _, err := pool.Exec(ctx, indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	return nil
}

// AnalyzeQuery provides EXPLAIN ANALYZE output for a query
func AnalyzeQuery(ctx context.Context, pool *pgxpool.Pool, query string, args ...interface{}) (string, error) {
	explainQuery := "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) " + query
	
	var result string
	err := pool.QueryRow(ctx, explainQuery, args...).Scan(&result)
	if err != nil {
		return "", fmt.Errorf("failed to analyze query: %w", err)
	}

	return result, nil
}
