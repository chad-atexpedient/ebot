# Performance & Scalability Guide

## Overview

This guide covers ebot's performance optimization features and best practices for scaling to enterprise workloads.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         Load Balancer                        │
└──────────────────────┬──────────────────────────────────────┘
                       │
       ┌───────────────┼───────────────┐
       │               │               │
┌──────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
│  ebot API   │ │  ebot API   │ │  ebot API   │
│  Server 1   │ │  Server 2   │ │  Server 3   │
└──────┬──────┘ └──────┬──────┘ └──────┬──────┘
       │               │               │
       └───────────────┼───────────────┘
                       │
         ┌─────────────┴─────────────┐
         │                           │
  ┌──────▼──────┐           ┌────────▼────────┐
  │    Redis    │           │   PostgreSQL    │
  │   Cache     │           │   (Primary +    │
  │             │           │    Replicas)    │
  └─────────────┘           └─────────────────┘
```

## Caching Strategy

### Redis Cache Layer

ebot uses Redis for high-performance caching:

```go
import "github.com/chad-atexpedient/ebot/pkg/cache"

// Create Redis cache
cacheManager, err := cache.NewRedisCacheManager(cache.RedisConfig{
    Address:  "localhost:6379",
    Password: "",
    DB:       0,
    PoolSize: 20,
})

// Use cache-aside pattern
servers, err := cacheManager.GetOrSet(ctx, 
    cache.MCPServerCacheKey(serverID),
    cache.MediumTTL,
    func() (interface{}, error) {
        // Fetch from database if not in cache
        return db.GetMCPServer(ctx, serverID)
    },
)
```

### Cache TTL Strategy

| Data Type | TTL | Reason |
|-----------|-----|--------|
| User Info | 15 minutes | Changes infrequently |
| MCP Server List | 5 minutes | Updates moderately |
| Tool List | 1 hour | Rarely changes |
| Quota Usage | 1 minute | Needs to be current |
| Cost Reports | 1 hour | Historical data |
| Model List | 24 hours | Very stable |

### Cache Invalidation

```go
// Invalidate specific key
cacheManager.Delete(ctx, cache.UserCacheKey(userID))

// Invalidate pattern
cacheManager.Invalidate(ctx, "user:*")

// Invalidate on write operations
func UpdateUser(ctx context.Context, userID string, updates User) error {
    if err := db.UpdateUser(ctx, userID, updates); err != nil {
        return err
    }
    
    // Invalidate cache
    return cacheManager.Delete(ctx, cache.UserCacheKey(userID))
}
```

## Database Optimization

### Connection Pooling

ebot uses optimized PostgreSQL connection pooling:

```go
import "github.com/chad-atexpedient/ebot/pkg/storage"

// For standard deployment
pool, err := storage.CreateOptimizedPool(ctx, connString, storage.DefaultPoolConfig())

// For enterprise high-load deployment
pool, err := storage.CreateOptimizedPool(ctx, connString, storage.EnterprisePoolConfig())
```

**Default Pool Config:**
- Max Connections: 25
- Min Connections: 5
- Max Lifetime: 1 hour
- Max Idle Time: 15 minutes

**Enterprise Pool Config:**
- Max Connections: 100
- Min Connections: 10
- Max Lifetime: 30 minutes
- Max Idle Time: 5 minutes

### Recommended Indexes

Run these to optimize query performance:

```sql
-- User indexes
CREATE INDEX CONCURRENTLY idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY idx_users_username ON users(username) WHERE deleted_at IS NULL;

-- MCP Server indexes
CREATE INDEX CONCURRENTLY idx_mcp_servers_user_id ON mcp_servers(user_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY idx_mcp_servers_status ON mcp_servers(status) WHERE deleted_at IS NULL;

-- Composite indexes for common patterns
CREATE INDEX CONCURRENTLY idx_mcp_servers_user_status ON mcp_servers(user_id, status) 
    WHERE deleted_at IS NULL;

-- Cost tracking indexes
CREATE INDEX CONCURRENTLY idx_llm_usage_user_id_timestamp ON llm_usage_events(user_id, timestamp DESC);
```

Or use the helper:

```go
err := storage.CreateRecommendedIndexes(ctx, pool)
```

### Query Optimization

**Use prepared statements:**

```go
// Good - uses prepared statement
rows, err := pool.Query(ctx, storage.ListMCPServersByUserQuery, userID, limit, offset)

// Bad - string concatenation
query := fmt.Sprintf("SELECT * FROM mcp_servers WHERE user_id = '%s'", userID)
```

**Analyze slow queries:**

```go
plan, err := storage.AnalyzeQuery(ctx, pool, 
    "SELECT * FROM mcp_servers WHERE user_id = $1", 
    userID,
)
fmt.Println(plan) // Shows execution plan
```

### Read Replicas

For read-heavy workloads, use read replicas:

```go
// Primary for writes
primaryPool, _ := pgxpool.New(ctx, primaryConnString)

// Replica for reads
replicaPool, _ := pgxpool.New(ctx, replicaConnString)

// Write to primary
_, err := primaryPool.Exec(ctx, "INSERT INTO users ...")

// Read from replica
rows, err := replicaPool.Query(ctx, "SELECT * FROM users WHERE ...")
```

## Response Compression

ebot automatically compresses responses > 1KB:

```go
import "github.com/chad-atexpedient/ebot/pkg/api/middleware"

// Add compression middleware
router.Use(middleware.CompressionMiddleware(1024)) // 1KB minimum

// Only compress specific content types
router.Use(middleware.ContentTypeCompressionMiddleware([]string{
    "application/json",
    "text/html",
}))
```

**Compression ratios:**
- JSON: 70-80% reduction
- HTML: 60-70% reduction
- Plain text: 50-60% reduction

## Load Testing

### Running Load Tests

```go
import "github.com/chad-atexpedient/ebot/tests/load"

// Basic load test (10 concurrent, 1000 requests)
result, err := load.BasicLoadTest(ctx, func(ctx context.Context) error {
    _, err := http.Get("http://localhost:8080/api/health")
    return err
})
result.PrintResults()

// Stress test (100 concurrent, 10000 requests)
result, err := load.StressTest(ctx, requestFunc)

// Spike test (instant 200 concurrent)
result, err := load.SpikeTest(ctx, requestFunc)

// Endurance test (25 concurrent for 30 minutes at 50 RPS)
result, err := load.EnduranceTest(ctx, requestFunc)
```

### Example Results

```
╔═══════════════════════════════════════════════════════╗
║           Load Test Results                           ║
╠═══════════════════════════════════════════════════════╣
║ Total Requests:              10000                    ║
║ Successful:                   9987 (99.9%)            ║
║ Failed:                         13 (0.1%)             ║
║ Duration:                     45.2s                   ║
║ Requests/sec:                221.24                   ║
╠═══════════════════════════════════════════════════════╣
║           Latency Statistics                          ║
╠═══════════════════════════════════════════════════════╣
║ Min:                         12ms                     ║
║ Average:                     45ms                     ║
║ Max:                        450ms                     ║
║ P50:                         42ms                     ║
║ P95:                         89ms                     ║
║ P99:                        156ms                     ║
╚═══════════════════════════════════════════════════════╝
```

## Performance Targets

### API Response Times
- **p50:** < 50ms
- **p95:** < 200ms
- **p99:** < 500ms

### Database Queries
- Simple queries: < 5ms
- Complex joins: < 50ms
- Aggregations: < 100ms

### Cache Performance
- Hit rate: > 80%
- Cache latency: < 1ms
- Miss latency: < 50ms (fetch from DB)

### Throughput
- API requests: 500+ RPS per server
- Database queries: 1000+ QPS
- Cache operations: 10,000+ OPS

## Scaling Guidelines

### Horizontal Scaling

**When to scale out:**
- CPU usage > 70% sustained
- Response time p95 > 200ms
- Connection pool saturation
- Queue depths increasing

**How to scale:**

```bash
# Kubernetes
kubectl scale deployment ebot --replicas=5

# Docker Compose
docker-compose up --scale ebot=5
```

### Vertical Scaling

**When to scale up:**
- Memory usage > 80%
- Database connection pool exhaustion
- Cache memory pressure

**Recommended specs:**

| Deployment Size | CPU | Memory | Database |
|-----------------|-----|--------|----------|
| Small (< 100 users) | 2 cores | 4 GB | 2 CPU, 8 GB |
| Medium (< 1000 users) | 4 cores | 8 GB | 4 CPU, 16 GB |
| Large (< 10k users) | 8 cores | 16 GB | 8 CPU, 32 GB |
| Enterprise (10k+ users) | 16+ cores | 32+ GB | 16+ CPU, 64+ GB |

## Monitoring Metrics

### Key Metrics to Track

**Application Metrics:**
```go
// Request rate
requests_total{method="GET", endpoint="/api/mcp-servers"}

// Response time
request_duration_seconds{endpoint="/api/threads"}

// Error rate
errors_total{type="timeout"}
```

**Cache Metrics:**
```go
// Cache hit rate
cache_hits_total / (cache_hits_total + cache_misses_total)

// Cache latency
cache_operation_duration_seconds
```

**Database Metrics:**
```go
// Connection pool usage
db_connections_active / db_connections_max

// Query performance
db_query_duration_seconds{query="list_servers"}
```

### Health Checks

```bash
# Overall health
curl http://localhost:8080/health

# Cache health
curl http://localhost:8080/health/cache

# Database health
curl http://localhost:8080/health/db

# Cluster health (HA)
curl http://localhost:8080/health/cluster
```

## Best Practices

### Do's ✅

1. **Enable caching** for all read operations
2. **Use connection pooling** with appropriate limits
3. **Add database indexes** for frequently queried fields
4. **Compress responses** > 1KB
5. **Monitor key metrics** continuously
6. **Load test** before major releases
7. **Use read replicas** for read-heavy workloads
8. **Implement circuit breakers** for downstream services

### Don'ts ❌

1. **Don't** bypass cache for every request
2. **Don't** use unbounded queries (always paginate)
3. **Don't** ignore slow query logs
4. **Don't** run without indexes in production
5. **Don't** set connection pool too high (causes contention)
6. **Don't** store large objects in cache
7. **Don't** forget to invalidate cache on updates
8. **Don't** skip load testing

## Troubleshooting Performance Issues

### Slow API Responses

1. Check cache hit rate:
   ```bash
   redis-cli INFO stats | grep keyspace_hits
   ```

2. Analyze slow queries:
   ```sql
   SELECT query, mean_exec_time, calls 
   FROM pg_stat_statements 
   ORDER BY mean_exec_time DESC 
   LIMIT 10;
   ```

3. Check connection pool:
   ```go
   stats := storage.GetPoolStats(pool)
   fmt.Printf("Active: %d/%d\n", stats.AcquiredConns, stats.MaxConns)
   ```

### High Memory Usage

1. Check cache size:
   ```bash
   redis-cli INFO memory
   ```

2. Reduce cache TTLs
3. Implement cache eviction policies
4. Check for memory leaks with pprof

### Database Bottlenecks

1. Enable query logging
2. Add missing indexes
3. Use read replicas
4. Increase connection pool size
5. Consider database sharding

## Performance Tuning Checklist

- [ ] Redis caching enabled
- [ ] Connection pool optimized
- [ ] Database indexes created
- [ ] Response compression enabled
- [ ] Load testing completed
- [ ] Monitoring dashboards configured
- [ ] Health checks implemented
- [ ] Circuit breakers configured
- [ ] Read replicas configured (if needed)
- [ ] Auto-scaling rules defined

## Expected Performance Improvements

After implementing all optimizations:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Average Response Time | 200ms | 20ms | **10x faster** |
| Throughput (RPS) | 50 | 500 | **10x more** |
| Database Load | 1000 QPS | 200 QPS | **5x reduction** |
| Cache Hit Rate | 0% | 85% | **85% fewer DB calls** |
| Memory Usage | 8 GB | 4 GB | **50% reduction** |
| Bandwidth | 100 MB/s | 30 MB/s | **70% reduction** |
| Cost per Request | $0.01 | $0.002 | **5x cheaper** |

## Next Steps

1. **Implement caching** - Start with high-traffic endpoints
2. **Optimize database** - Add recommended indexes
3. **Load test** - Establish baselines
4. **Monitor** - Set up dashboards and alerts
5. **Iterate** - Continuously measure and improve

---

**Questions?** See [ROADMAP.md](ROADMAP.md) or reach out to the team.
