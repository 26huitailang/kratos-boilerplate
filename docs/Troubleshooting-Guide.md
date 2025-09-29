# Troubleshooting Guide

## Table of Contents
- [Quick Diagnostic Checklist](#quick-diagnostic-checklist)
- [Common Issues](#common-issues)
- [Application Logs](#application-logs)
- [Health Check Issues](#health-check-issues)
- [Database Problems](#database-problems)
- [Performance Issues](#performance-issues)
- [Emergency Procedures](#emergency-procedures)

## Quick Diagnostic Checklist

### 1. Service Status Check
```bash
# Check application status
curl http://localhost:8000/health

# Check process status
ps aux | grep kratos-boilerplate

# Check ports
netstat -tlnp | grep 8000
```

### 2. Basic Health Verification
```bash
# HTTP health check
curl -f http://localhost:8000/health || echo "Health check failed"

# gRPC health check
grpc_health_probe -addr=localhost:9000

# Database connectivity
curl http://localhost:8000/health/database
```

## Common Issues

### Application Won't Start

**Symptoms:**
- Process exits immediately
- "Port already in use" error
- Configuration validation errors

**Solutions:**
```bash
# Check port usage
lsof -i :8000
kill -9 <PID>

# Validate configuration
./kratos-boilerplate -conf=configs/config.yaml -validate

# Check logs
tail -f logs/application.log
```

### High Memory Usage

**Symptoms:**
- Memory usage > 90%
- OOM kills in logs
- Slow response times

**Immediate Actions:**
```bash
# Check memory usage
free -h
ps aux --sort=-%mem | head -10

# Restart service if critical
systemctl restart kratos-boilerplate

# Monitor memory in real-time
watch -n 1 'free -h && ps aux | grep kratos-boilerplate'
```

### Database Connection Issues

**Symptoms:**
- "connection refused" errors
- Timeout errors
- Transaction deadlocks

**Solutions:**
```bash
# Test database connectivity
mysql -h localhost -u user -p database_name

# Check connection pool
curl http://localhost:8000/health/database

# Reset connections
systemctl restart kratos-boilerplate
```

## Application Logs

### Log Locations
```
Production: /var/log/kratos-boilerplate/
Development: ./logs/
Docker: docker logs <container_id>
```

### Key Log Patterns
```bash
# Error tracking
grep -i "error\|fatal\|panic" /var/log/kratos-boilerplate/app.log

# Performance issues
grep -i "slow\|timeout\|latency" /var/log/kratos-boilerplate/app.log

# Database issues
grep -i "database\|sql\|connection" /var/log/kratos-boilerplate/app.log
```

## Health Check Issues

### Health Endpoint Returns 500
```bash
# Check individual components
curl http://localhost:8000/health/database
curl http://localhost:8000/health/cache
curl http://localhost:8000/health/external

# Check detailed status
curl -v http://localhost:8000/health
```

### gRPC Health Check Fails
```bash
# Install grpc_health_probe
wget https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.13/grpc_health_probe-linux-amd64
chmod +x grpc_health_probe-linux-amd64

# Test health check
./grpc_health_probe-linux-amd64 -addr=localhost:9000
```

## Database Problems

### Connection Pool Exhaustion
```bash
# Check active connections
SHOW PROCESSLIST;

# Reset connection pool
systemctl restart kratos-boilerplate

# Increase pool size in config
# database.max_open_conns: 50
# database.max_idle_conns: 10
```

### Slow Queries
```sql
-- Enable slow query log
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 2;

-- Check slow queries
SHOW VARIABLES LIKE 'slow_query_log%';
```

## Performance Issues

### High CPU Usage
```bash
# Identify CPU-intensive processes
top -p $(pgrep kratos-boilerplate)

# CPU profiling
curl http://localhost:8000/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Check for infinite loops
strace -p <PID> -c
```

### High Latency
```bash
# Check response times
curl -w "@curl-format.txt" http://localhost:8000/api/v1/endpoint

# Create curl-format.txt:
# time_namelookup:  %{time_namelookup}s\n
# time_connect:     %{time_connect}s\n
# time_total:       %{time_total}s\n
```

## Emergency Procedures

### Service Restart
```bash
# Graceful restart
systemctl reload kratos-boilerplate

# Force restart
systemctl restart kratos-boilerplate

# Emergency stop
systemctl stop kratos-boilerplate
pkill -f kratos-boilerplate
```

### Emergency Scaling
```bash
# Docker Compose
docker-compose up --scale app=3

# Kubernetes
kubectl scale deployment kratos-boilerplate --replicas=3
```

### Rollback Procedure
```bash
# Docker
docker tag kratos-boilerplate:previous kratos-boilerplate:latest
docker-compose up -d

# Kubernetes
kubectl rollout undo deployment/kratos-boilerplate
kubectl rollout status deployment/kratos-boilerplate
```

## Contact Information

**On-Call Engineer:** [Contact Details]
**Escalation Path:** [Manager Contact]
**Emergency Hotline:** [Phone Number]

---

**Last Updated:** 2025-09-28
**Version:** 1.0
**Maintained By:** DevOps Team