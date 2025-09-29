# Health Check and Monitoring

## Overview

This document describes the health check and monitoring capabilities implemented in the kratos-boilerplate project.

## Health Check Endpoints

The application provides several health check endpoints for different purposes:

### Primary Health Check
```
GET /health
```

Returns the overall health status of the application including all registered health checks.

**Response Example:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-28T18:30:00Z",
  "duration": "5ms",
  "checks": {
    "database": {
      "name": "database",
      "status": "healthy",
      "message": "Database connection successful",
      "timestamp": "2025-09-28T18:30:00Z",
      "duration": "2ms"
    },
    "redis": {
      "name": "redis",
      "status": "healthy", 
      "message": "Cache connection successful",
      "timestamp": "2025-09-28T18:30:00Z",
      "duration": "1ms"
    },
    "memory": {
      "name": "memory",
      "status": "healthy",
      "message": "Memory usage is within acceptable limits",
      "timestamp": "2025-09-28T18:30:00Z", 
      "duration": "1ms"
    }
  }
}
```

### Kubernetes Probes

#### Liveness Probe
```
GET /health/live
```

Used by Kubernetes to determine if the container should be restarted. This endpoint checks basic application functionality.

#### Readiness Probe  
```
GET /health/ready
```

Used by Kubernetes to determine if the container is ready to receive traffic. This endpoint verifies that all dependencies are healthy.

## Health Check Status Codes

- **200 OK**: Service is healthy or degraded
- **503 Service Unavailable**: Service is unhealthy

## Health Check Components

The health check system includes the following components:

### 1. Database Health Check
- **Component**: PostgreSQL database
- **Check**: Connection ping with timeout
- **Threshold**: Response time < 5s

### 2. Cache Health Check  
- **Component**: Redis cache
- **Check**: Connection ping with timeout
- **Threshold**: Response time < 3s

### 3. Memory Health Check
- **Component**: Application memory usage
- **Check**: Memory usage percentage
- **Threshold**: < 85% of available memory

### 4. Liveness Check
- **Component**: Basic application functionality
- **Check**: Service responsiveness
- **Purpose**: Container restart indicator

## Metrics Endpoint

### Basic Metrics
```
GET /metrics
```

Returns basic application metrics in JSON format.

**Response Example:**
```json
{
  "uptime": "2h15m30s",
  "goroutines": 45,
  "memory": {
    "allocated_mb": 12.5,
    "total_allocated_mb": 125.7,
    "system_mb": 18.2,
    "gc_count": 15
  },
  "timestamp": "2025-09-28T18:30:00Z"
}
```

### Metrics Description

- **uptime**: Application uptime since last restart
- **goroutines**: Current number of active goroutines
- **memory.allocated_mb**: Currently allocated memory in MB
- **memory.total_allocated_mb**: Total memory allocated since start
- **memory.system_mb**: Memory obtained from the system
- **memory.gc_count**: Number of garbage collection cycles

## Configuration

### Development Environment
```yaml
monitoring:
  metrics:
    enabled: true
    path: "/metrics"
  health:
    enabled: true
    path: "/health"
```

### Production Environment
```yaml
monitoring:
  metrics:
    enabled: "${METRICS_ENABLED:true}"
    addr: "${METRICS_ADDR:0.0.0.0:8080}"
    path: "${METRICS_PATH:/metrics}"
  health:
    enabled: "${HEALTH_ENABLED:true}"
    addr: "${HEALTH_ADDR:0.0.0.0:8081}"
    path: "${HEALTH_PATH:/health}"
  tracing:
    enabled: "${TRACING_ENABLED:true}"
    endpoint: "${TRACING_ENDPOINT}"
    service_name: "${TRACING_SERVICE_NAME:kratos-boilerplate}"
```

## Docker Compose Health Checks

The application is configured for Docker Compose health checks:

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8000/health"]
  interval: 10s
  timeout: 5s
  retries: 10
  start_period: 30s
```

## Kubernetes Health Checks

### Deployment Configuration
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kratos-boilerplate
spec:
  template:
    spec:
      containers:
      - name: app
        image: kratos-boilerplate:latest
        ports:
        - containerPort: 8000
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
```

## Monitoring Integration

### Prometheus Integration
The metrics endpoint can be scraped by Prometheus for monitoring:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'kratos-boilerplate'
    static_configs:
      - targets: ['localhost:8000']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Grafana Dashboard
Basic metrics can be visualized in Grafana:

- Application uptime
- Memory usage trends
- Goroutine count
- Health check status

## Best Practices

### Health Check Design
1. **Fast Response**: Health checks should complete in < 5 seconds
2. **Dependency Validation**: Verify critical dependencies (database, cache)
3. **Graceful Degradation**: Return appropriate status for partial failures
4. **Resource Monitoring**: Include memory and CPU usage checks

### Monitoring Strategy  
1. **Metrics Collection**: Collect key performance indicators
2. **Alerting**: Set up alerts for critical thresholds
3. **Logging Integration**: Correlate metrics with application logs
4. **Trend Analysis**: Monitor trends over time

### Production Deployment
1. **Separate Endpoints**: Use different ports for metrics and health checks
2. **Security**: Restrict access to monitoring endpoints
3. **Load Balancer Integration**: Configure load balancer health checks
4. **Auto-scaling**: Use health checks for auto-scaling decisions

## Troubleshooting

### Common Issues

#### Unhealthy Database Status
```bash
# Check database connectivity
curl http://localhost:8000/health | jq '.checks.database'

# Verify database configuration
psql -h localhost -U postgres -d test_db -c "SELECT 1;"
```

#### High Memory Usage
```bash
# Check memory metrics
curl http://localhost:8000/metrics | jq '.memory'

# Monitor memory usage over time
watch -n 5 'curl -s http://localhost:8000/metrics | jq .memory'
```

#### Service Unavailable
```bash
# Check overall health status
curl -v http://localhost:8000/health

# Check individual components
curl http://localhost:8000/health | jq '.checks'
```

## Future Enhancements

### Planned Features
1. **Custom Metrics**: Application-specific business metrics
2. **Advanced Alerting**: Integration with alerting systems
3. **Performance Profiling**: CPU and memory profiling endpoints  
4. **Distributed Tracing**: Integration with Jaeger/Zipkin
5. **SLA Monitoring**: Response time and availability tracking

### Integration Roadmap
1. **Phase 1**: Basic health checks and metrics ✅
2. **Phase 2**: Prometheus integration
3. **Phase 3**: Grafana dashboards
4. **Phase 4**: Advanced monitoring and alerting
5. **Phase 5**: Performance optimization based on metrics