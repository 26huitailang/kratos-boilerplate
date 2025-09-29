# 运维手册

## 概述

本手册提供 kratos-boilerplate 项目的日常运维指导，包括监控、维护、故障处理和性能优化等关键运维任务。

## 目录

- [日常监控](#日常监控)
- [系统维护](#系统维护)
- [性能监控](#性能监控)
- [日志管理](#日志管理)
- [备份策略](#备份策略)
- [安全管理](#安全管理)
- [容量规划](#容量规划)
- [故障响应](#故障响应)
- [变更管理](#变更管理)
- [SLA 和指标](#sla-和指标)

## 日常监控

### 关键指标监控

#### 应用指标
- **健康状态**: `/health` 端点响应
- **响应时间**: API 请求平均响应时间
- **错误率**: 4xx/5xx 错误比例
- **吞吐量**: 每秒请求数 (RPS)
- **活跃连接**: 并发连接数

#### 系统指标
- **CPU 使用率**: 保持在 70% 以下
- **内存使用率**: 保持在 80% 以下
- **磁盘使用率**: 保持在 85% 以下
- **网络流量**: 入站/出站流量监控

#### 数据库指标
- **连接数**: 活跃连接数
- **查询性能**: 慢查询监控
- **锁等待**: 死锁和锁等待时间
- **存储空间**: 数据库大小增长

#### Redis 指标
- **内存使用**: Redis 内存占用
- **命中率**: 缓存命中率
- **连接数**: 客户端连接数
- **过期键**: 过期键处理

### 监控工具配置

#### Prometheus 配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "kratos_rules.yml"

scrape_configs:
  - job_name: 'kratos-boilerplate'
    static_configs:
      - targets: ['app:8000']
    metrics_path: '/metrics'
    scrape_interval: 10s
    
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']
      
  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

#### 告警规则

```yaml
# kratos_rules.yml
groups:
  - name: kratos-boilerplate
    rules:
      # 应用健康检查
      - alert: ApplicationDown
        expr: up{job="kratos-boilerplate"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "应用服务不可用"
          description: "kratos-boilerplate 服务已停止响应超过 1 分钟"

      # 高错误率
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "高错误率告警"
          description: "5xx 错误率超过 5%，当前值: {{ $value }}"

      # 响应时间过长
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 3m
        labels:
          severity: warning
        annotations:
          summary: "响应时间过长"
          description: "95% 请求响应时间超过 1 秒，当前值: {{ $value }}s"

      # 内存使用率过高
      - alert: HighMemoryUsage
        expr: (process_resident_memory_bytes / container_spec_memory_limit_bytes) > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "内存使用率过高"
          description: "内存使用率超过 80%，当前值: {{ $value }}"

      # 数据库连接数过多
      - alert: HighDatabaseConnections
        expr: pg_stat_database_numbackends > 80
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "数据库连接数过多"
          description: "数据库连接数超过 80，当前值: {{ $value }}"
```

#### Grafana 仪表板

```json
{
  "dashboard": {
    "title": "Kratos Boilerplate 监控仪表板",
    "panels": [
      {
        "title": "请求率",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "响应时间",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ]
      },
      {
        "title": "错误率",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"4..\"}[5m])",
            "legendFormat": "4xx errors"
          },
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m])",
            "legendFormat": "5xx errors"
          }
        ]
      }
    ]
  }
}
```

### 监控检查清单

#### 每日检查
- [ ] 检查应用健康状态
- [ ] 查看错误日志
- [ ] 监控资源使用率
- [ ] 检查备份状态
- [ ] 验证监控告警

#### 每周检查
- [ ] 审查性能趋势
- [ ] 检查存储空间增长
- [ ] 更新安全补丁
- [ ] 清理旧日志文件
- [ ] 验证备份恢复

#### 每月检查
- [ ] 容量规划评估
- [ ] 安全漏洞扫描
- [ ] 性能基准测试
- [ ] 更新监控规则
- [ ] 灾难恢复演练

## 系统维护

### 例行维护任务

#### 应用维护

```bash
# 1. 检查应用状态
curl -f http://localhost:8000/health || echo "健康检查失败"

# 2. 检查应用版本
curl -s http://localhost:8000/version

# 3. 重启应用 (如果需要)
docker-compose restart app

# 4. 检查应用日志
docker-compose logs --tail=100 app
```

#### 数据库维护

```sql
-- 1. 检查数据库大小
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname NOT IN ('information_schema', 'pg_catalog')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 2. 检查连接数
SELECT count(*) as active_connections 
FROM pg_stat_activity 
WHERE state = 'active';

-- 3. 查看慢查询
SELECT query, mean_time, calls, total_time 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- 4. 更新统计信息
ANALYZE;

-- 5. 清理过期数据 (示例)
DELETE FROM operation_logs 
WHERE created_at < NOW() - INTERVAL '30 days';
```

#### Redis 维护

```bash
# 1. 检查 Redis 状态
redis-cli ping

# 2. 查看内存使用
redis-cli info memory

# 3. 查看连接数
redis-cli info clients

# 4. 清理过期键
redis-cli --scan --pattern "*" | xargs redis-cli del

# 5. 持久化数据
redis-cli bgsave
```

### 更新和升级

#### 应用更新流程

```bash
# 1. 备份当前版本
docker tag kratos-boilerplate:latest kratos-boilerplate:backup-$(date +%Y%m%d)

# 2. 拉取新版本
docker pull kratos-boilerplate:v1.2.0

# 3. 更新配置 (如果需要)
cp config.yaml config.yaml.backup
# 编辑新配置

# 4. 执行数据库迁移
migrate -path ./migrations -database $DATABASE_URL up

# 5. 滚动更新
docker-compose up -d app

# 6. 验证新版本
curl -f http://localhost:8000/health
curl -s http://localhost:8000/version

# 7. 回滚 (如果需要)
# docker-compose down
# docker tag kratos-boilerplate:backup-$(date +%Y%m%d) kratos-boilerplate:latest
# docker-compose up -d app
```

#### 系统升级

```bash
# 1. 更新系统包
sudo apt update && sudo apt upgrade -y

# 2. 重启服务器 (如果需要)
sudo reboot

# 3. 验证服务
systemctl status nginx
systemctl status postgresql
systemctl status redis

# 4. 检查应用
docker-compose ps
curl -f http://localhost:8000/health
```

## 性能监控

### 性能指标收集

#### 应用性能

```go
// 在应用中添加性能指标
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "Duration of HTTP requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
)
```

#### 性能基准测试

```bash
# 1. 压力测试
ab -n 10000 -c 100 http://localhost:8000/api/helloworld/world

# 2. 使用 wrk
wrk -t12 -c400 -d30s http://localhost:8000/api/helloworld/world

# 3. API 负载测试
echo "GET http://localhost:8000/api/auth/health" | vegeta attack -duration=30s -rate=1000 | vegeta report
```

#### 数据库性能

```sql
-- 查看最慢的查询
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    max_time
FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 10;

-- 查看表统计信息
SELECT 
    relname as table_name,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch
FROM pg_stat_user_tables 
ORDER BY seq_scan DESC;

-- 查看索引使用情况
SELECT 
    indexrelname as index_name,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes 
ORDER BY idx_scan DESC;
```

### 性能优化建议

#### 应用层优化

1. **连接池优化**
```yaml
# 数据库连接池配置
database:
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: 300s
```

2. **缓存策略**
```yaml
# Redis 缓存配置
redis:
  pool_size: 20
  min_idle_conns: 5
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
```

3. **HTTP 优化**
```yaml
# HTTP 服务配置
server:
  http:
    timeout: 30s
    read_timeout: 10s
    write_timeout: 10s
    idle_timeout: 60s
```

#### 数据库优化

```sql
-- 创建适当的索引
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE INDEX CONCURRENTLY idx_logs_created_at ON operation_logs(created_at);

-- 分区大表
CREATE TABLE operation_logs_2024 PARTITION OF operation_logs 
FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');

-- 定期维护
REINDEX INDEX CONCURRENTLY idx_users_email;
VACUUM ANALYZE users;
```

## 日志管理

### 日志级别和格式

#### 应用日志配置

```yaml
# config.yaml
log:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "stdout"  # stdout, file
  file:
    path: "/app/logs/app.log"
    max_size: 100  # MB
    max_backups: 10
    max_age: 30  # days
    compress: true
```

#### 日志轮转配置

```bash
# /etc/logrotate.d/kratos-boilerplate
/var/log/kratos-boilerplate/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 kratos kratos
    postrotate
        systemctl reload supervisor
    endscript
}
```

### 日志聚合和分析

#### ELK Stack 配置

```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/kratos-boilerplate/*.log
  fields:
    service: kratos-boilerplate
    environment: production
  fields_under_root: true
  json.keys_under_root: true
  json.add_error_key: true

output.logstash:
  hosts: ["logstash:5044"]

processors:
  - add_host_metadata:
      when.not.contains.tags: forwarded
```

```ruby
# logstash.conf
input {
  beats {
    port => 5044
  }
}

filter {
  if [service] == "kratos-boilerplate" {
    date {
      match => [ "timestamp", "ISO8601" ]
    }
    
    if [level] == "error" {
      mutate {
        add_tag => ["error"]
      }
    }
  }
}

output {
  elasticsearch {
    hosts => ["elasticsearch:9200"]
    index => "kratos-boilerplate-%{+YYYY.MM.dd}"
  }
}
```

### 日志查询和分析

#### 常用查询

```bash
# 查看错误日志
grep -i error /var/log/kratos-boilerplate/app.log | tail -20

# 查看最近的访问日志
tail -f /var/log/nginx/access.log

# 分析错误趋势
awk '/ERROR/ {print $1, $2}' /var/log/kratos-boilerplate/app.log | sort | uniq -c

# 查看特定时间段的日志
sed -n '/2024-01-01 10:00/,/2024-01-01 11:00/p' /var/log/kratos-boilerplate/app.log
```

#### Elasticsearch 查询

```json
{
  "query": {
    "bool": {
      "filter": [
        {"term": {"service": "kratos-boilerplate"}},
        {"term": {"level": "error"}},
        {"range": {
          "@timestamp": {
            "gte": "now-1h"
          }
        }}
      ]
    }
  },
  "aggs": {
    "error_count": {
      "date_histogram": {
        "field": "@timestamp",
        "interval": "5m"
      }
    }
  }
}
```

## 备份策略

### 数据库备份

#### 自动备份脚本

```bash
#!/bin/bash
# /opt/scripts/backup_database.sh

set -e

# 配置
BACKUP_DIR="/backups/postgresql"
DATABASE_URL="postgresql://user:password@localhost:5432/kratos_prod"
RETENTION_DAYS=30
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 执行备份
echo "开始数据库备份: $DATE"
pg_dump $DATABASE_URL | gzip > "$BACKUP_DIR/kratos_backup_$DATE.sql.gz"

# 验证备份
if [ $? -eq 0 ]; then
    echo "备份成功: $BACKUP_DIR/kratos_backup_$DATE.sql.gz"
else
    echo "备份失败!"
    exit 1
fi

# 清理旧备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete

echo "备份完成"
```

#### 定时备份

```bash
# 添加到 crontab
crontab -e

# 每日凌晨 2 点备份
0 2 * * * /opt/scripts/backup_database.sh >> /var/log/backup.log 2>&1

# 每周日凌晨 1 点进行完整备份
0 1 * * 0 /opt/scripts/full_backup.sh >> /var/log/backup.log 2>&1
```

### 文件备份

```bash
#!/bin/bash
# 应用文件备份

BACKUP_DIR="/backups/files"
APP_DIR="/opt/kratos-boilerplate"
DATE=$(date +%Y%m%d_%H%M%S)

# 备份配置文件
tar -czf "$BACKUP_DIR/config_backup_$DATE.tar.gz" \
    -C "$APP_DIR" configs/

# 备份日志文件
tar -czf "$BACKUP_DIR/logs_backup_$DATE.tar.gz" \
    -C "/var/log" kratos-boilerplate/

# 备份用户上传文件
tar -czf "$BACKUP_DIR/uploads_backup_$DATE.tar.gz" \
    -C "$APP_DIR" uploads/
```

### 恢复流程

#### 数据库恢复

```bash
#!/bin/bash
# 数据库恢复脚本

BACKUP_FILE=$1
DATABASE_URL="postgresql://user:password@localhost:5432/kratos_prod"

if [ -z "$BACKUP_FILE" ]; then
    echo "用法: $0 <backup_file.sql.gz>"
    exit 1
fi

echo "开始恢复数据库: $BACKUP_FILE"

# 停止应用
docker-compose stop app

# 恢复数据库
gunzip -c $BACKUP_FILE | psql $DATABASE_URL

if [ $? -eq 0 ]; then
    echo "数据库恢复成功"
else
    echo "数据库恢复失败!"
    exit 1
fi

# 重启应用
docker-compose start app

echo "恢复完成"
```

## 安全管理

### 安全监控

#### 访问日志分析

```bash
# 检查异常访问
awk '{print $1}' /var/log/nginx/access.log | sort | uniq -c | sort -nr | head -20

# 检查 4xx/5xx 错误
awk '$9 >= 400 {print $9, $7}' /var/log/nginx/access.log | sort | uniq -c

# 检查可疑 User-Agent
awk -F'"' '{print $6}' /var/log/nginx/access.log | sort | uniq -c | sort -nr
```

#### 安全告警

```yaml
# 安全告警规则
groups:
  - name: security
    rules:
      - alert: TooManyFailedLogins
        expr: increase(failed_login_attempts_total[5m]) > 10
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "登录失败次数过多"
          
      - alert: UnusualTrafficPattern
        expr: rate(http_requests_total[5m]) > 1000
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "异常流量模式"
```

### 证书管理

#### SSL 证书更新

```bash
#!/bin/bash
# SSL 证书自动更新

# 使用 Let's Encrypt
certbot renew --nginx --quiet

# 验证证书
openssl x509 -in /etc/letsencrypt/live/yourdomain.com/cert.pem -text -noout | grep "Not After"

# 重新加载 Nginx
nginx -t && systemctl reload nginx
```

#### 证书监控

```bash
# 检查证书过期时间
echo | openssl s_client -servername yourdomain.com -connect yourdomain.com:443 2>/dev/null | openssl x509 -noout -dates
```

### 漏洞扫描

```bash
# 系统漏洞扫描
apt list --upgradable | grep -i security

# 容器安全扫描
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
    -v $PWD:/root/.cache/ aquasec/trivy image kratos-boilerplate:latest

# 依赖漏洞扫描
go list -json -m all | nancy sleuth
```

## 容量规划

### 资源监控

#### 趋势分析

```bash
# CPU 使用趋势 (过去7天)
sar -u 1 1 | grep Average

# 内存使用趋势
free -h && cat /proc/meminfo

# 磁盘使用增长
df -h | grep -v tmpfs

# 网络流量统计
sar -n DEV 1 1 | grep Average
```

#### 容量预测

```python
# 简单的容量预测脚本
import pandas as pd
import numpy as np
from sklearn.linear_model import LinearRegression

# 读取历史数据
df = pd.read_csv('metrics.csv')
df['date'] = pd.to_datetime(df['date'])
df['days'] = (df['date'] - df['date'].min()).dt.days

# 训练模型
X = df[['days']].values
y = df['cpu_usage'].values

model = LinearRegression()
model.fit(X, y)

# 预测未来30天
future_days = np.array([[df['days'].max() + 30]])
predicted_usage = model.predict(future_days)

print(f"30天后预测CPU使用率: {predicted_usage[0]:.2f}%")
```

### 扩容策略

#### 水平扩容

```yaml
# Kubernetes HPA 配置
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: kratos-boilerplate-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: kratos-boilerplate
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

#### 垂直扩容

```yaml
# 增加资源限制
resources:
  limits:
    memory: "2Gi"
    cpu: "2000m"
  requests:
    memory: "1Gi"
    cpu: "1000m"
```

## 故障响应

### 故障分级

#### 级别定义

1. **P0 (严重)**: 服务完全不可用
2. **P1 (高)**: 核心功能受影响
3. **P2 (中)**: 部分功能受影响
4. **P3 (低)**: 性能下降或非核心功能

#### 响应时间

- **P0**: 15分钟内响应，1小时内解决
- **P1**: 30分钟内响应，4小时内解决
- **P2**: 2小时内响应，24小时内解决
- **P3**: 24小时内响应，72小时内解决

### 故障处理流程

#### 快速响应检查清单

```bash
# 1. 确认故障范围
curl -f http://localhost:8000/health
curl -f http://localhost:8000/metrics

# 2. 检查应用状态
docker-compose ps
kubectl get pods -n kratos-boilerplate

# 3. 查看最近日志
docker-compose logs --tail=50 app
kubectl logs --tail=50 deployment/kratos-boilerplate -n kratos-boilerplate

# 4. 检查资源使用
top
free -h
df -h

# 5. 检