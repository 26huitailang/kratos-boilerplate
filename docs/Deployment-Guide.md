# 部署指南

## 概述

本文档提供了 kratos-boilerplate 项目的完整部署指南，包括开发环境、测试环境和生产环境的部署方案。

## 目录

- [环境要求](#环境要求)
- [快速部署](#快速部署)
- [开发环境部署](#开发环境部署)
- [测试环境部署](#测试环境部署)
- [生产环境部署](#生产环境部署)
- [容器化部署](#容器化部署)
- [Kubernetes 部署](#kubernetes-部署)
- [配置管理](#配置管理)
- [数据库迁移](#数据库迁移)
- [监控和日志](#监控和日志)

## 环境要求

### 基础环境
- **Go**: 1.19+
- **Node.js**: 16+ (用于前端构建)
- **Docker**: 20.0+ (用于容器化部署)
- **Docker Compose**: 2.0+

### 数据库
- **PostgreSQL**: 13+ (主数据库)
- **Redis**: 6.0+ (缓存和会话存储)

### 可选组件
- **Kubernetes**: 1.20+ (生产环境容器编排)
- **Nginx**: 1.18+ (反向代理)
- **Prometheus**: 2.0+ (监控)
- **Grafana**: 8.0+ (可视化)

## 快速部署

### 使用 Docker Compose

```bash
# 1. 克隆项目
git clone <repository-url>
cd kratos-boilerplate

# 2. 复制并配置环境变量
cp .env.example .env
# 编辑 .env 文件设置必要的环境变量

# 3. 启动所有服务
docker-compose up -d

# 4. 检查服务状态
docker-compose ps

# 5. 查看日志
docker-compose logs -f app
```

### 验证部署
```bash
# 检查健康状态
curl http://localhost:8000/health

# 检查 API
curl http://localhost:8000/helloworld/world
```

## 开发环境部署

### 本地开发设置

1. **安装依赖**
```bash
# 安装 Go 依赖
go mod download

# 安装前端依赖
cd frontend && npm install
```

2. **配置数据库**
```bash
# 启动 PostgreSQL (使用 Docker)
docker run -d \
  --name postgres-dev \
  -e POSTGRES_DB=kratos_dev \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:13

# 启动 Redis
docker run -d \
  --name redis-dev \
  -p 6379:6379 \
  redis:6.0
```

3. **运行数据库迁移**
```bash
# 安装迁移工具
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 运行迁移
migrate -path ./migrations -database "postgresql://postgres:postgres@localhost:5432/kratos_dev?sslmode=disable" up
```

4. **启动应用**
```bash
# 启动后端
go run ./cmd/kratos-boilerplate -conf ./configs/config.yaml

# 启动前端 (新终端)
cd frontend && npm run dev
```

### 开发工具

```bash
# 代码生成
make generate

# 运行测试
make test

# 代码检查
make lint

# 构建
make build
```

## 测试环境部署

### 使用 Docker Compose

1. **创建测试环境配置**
```yaml
# docker-compose.test.yml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      - ENV=test
      - DATABASE_URL=postgresql://postgres:postgres@db:5432/kratos_test?sslmode=disable
      - REDIS_URL=redis://redis:6379
    ports:
      - "8001:8000"
    depends_on:
      - db
      - redis

  db:
    image: postgres:13
    environment:
      POSTGRES_DB: kratos_test
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    volumes:
      - postgres_test_data:/var/lib/postgresql/data

  redis:
    image: redis:6.0
    volumes:
      - redis_test_data:/data

volumes:
  postgres_test_data:
  redis_test_data:
```

2. **部署测试环境**
```bash
# 构建并启动
docker-compose -f docker-compose.test.yml up -d

# 运行数据库迁移
docker-compose -f docker-compose.test.yml exec app \
  migrate -path ./migrations -database $DATABASE_URL up

# 运行测试
docker-compose -f docker-compose.test.yml exec app go test ./...
```

## 生产环境部署

### 传统部署方式

1. **服务器准备**
```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装必要软件
sudo apt install -y nginx postgresql redis-server supervisor

# 创建应用用户
sudo useradd -r -s /bin/false kratos
sudo mkdir -p /opt/kratos-boilerplate
sudo chown kratos:kratos /opt/kratos-boilerplate
```

2. **数据库设置**
```bash
# 配置 PostgreSQL
sudo -u postgres createuser kratos
sudo -u postgres createdb kratos_prod -O kratos
sudo -u postgres psql -c "ALTER USER kratos PASSWORD 'your-secure-password';"

# 配置 Redis
sudo systemctl enable redis-server
sudo systemctl start redis-server
```

3. **应用部署**
```bash
# 上传应用文件
scp -r ./build/* user@server:/opt/kratos-boilerplate/
scp ./configs/production.yaml user@server:/opt/kratos-boilerplate/config.yaml

# 设置权限
sudo chown -R kratos:kratos /opt/kratos-boilerplate
sudo chmod +x /opt/kratos-boilerplate/kratos-boilerplate
```

4. **系统服务配置**
```ini
# /etc/supervisor/conf.d/kratos-boilerplate.conf
[program:kratos-boilerplate]
command=/opt/kratos-boilerplate/kratos-boilerplate -conf /opt/kratos-boilerplate/config.yaml
directory=/opt/kratos-boilerplate
user=kratos
autostart=true
autorestart=true
stderr_logfile=/var/log/kratos-boilerplate/error.log
stdout_logfile=/var/log/kratos-boilerplate/access.log
environment=ENV=production
```

5. **Nginx 配置**
```nginx
# /etc/nginx/sites-available/kratos-boilerplate
server {
    listen 80;
    server_name your-domain.com;

    # SSL 重定向
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/your/cert.pem;
    ssl_certificate_key /path/to/your/key.pem;

    # 安全头
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains";

    # API 代理
    location /api/ {
        proxy_pass http://localhost:8000/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 静态文件
    location / {
        root /opt/kratos-boilerplate/frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    # 健康检查
    location /health {
        proxy_pass http://localhost:8000/health;
        access_log off;
    }
}
```

## 容器化部署

### Dockerfile

```dockerfile
# 多阶段构建
FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o kratos-boilerplate ./cmd/kratos-boilerplate

# 运行时镜像
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/kratos-boilerplate .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/migrations ./migrations

EXPOSE 8000

CMD ["./kratos-boilerplate", "-conf", "./configs/config.yaml"]
```

### Docker Compose 生产配置

```yaml
version: '3.8'

services:
  app:
    image: kratos-boilerplate:latest
    restart: unless-stopped
    environment:
      - ENV=production
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - JWT_SECRET_KEY=${JWT_SECRET_KEY}
    ports:
      - "8000:8000"
    depends_on:
      - db
      - redis
    volumes:
      - ./logs:/app/logs
    networks:
      - kratos-net

  db:
    image: postgres:13
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backups:/backups
    networks:
      - kratos-net

  redis:
    image: redis:6.0
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    networks:
      - kratos-net

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
      - ./frontend/dist:/usr/share/nginx/html
    depends_on:
      - app
    networks:
      - kratos-net

volumes:
  postgres_data:
  redis_data:

networks:
  kratos-net:
    driver: bridge
```

## Kubernetes 部署

### 命名空间和配置

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kratos-boilerplate

---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: kratos-boilerplate
data:
  config.yaml: |
    server:
      http:
        addr: 0.0.0.0:8000
        timeout: 30s
      grpc:
        addr: 0.0.0.0:9000
        timeout: 30s
    # ... 其他配置
```

### 密钥管理

```yaml
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
  namespace: kratos-boilerplate
type: Opaque
data:
  database-url: <base64-encoded-database-url>
  redis-url: <base64-encoded-redis-url>
  jwt-secret: <base64-encoded-jwt-secret>
```

### 应用部署

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kratos-boilerplate
  namespace: kratos-boilerplate
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kratos-boilerplate
  template:
    metadata:
      labels:
        app: kratos-boilerplate
    spec:
      containers:
      - name: app
        image: kratos-boilerplate:latest
        ports:
        - containerPort: 8000
        env:
        - name: ENV
          value: "production"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: redis-url
        - name: JWT_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: jwt-secret
        volumeMounts:
        - name: config
          mountPath: /app/configs
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          limits:
            memory: "1Gi"
            cpu: "1000m"
          requests:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: config
        configMap:
          name: app-config

---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: kratos-boilerplate-service
  namespace: kratos-boilerplate
spec:
  selector:
    app: kratos-boilerplate
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8000
  type: ClusterIP

---
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: kratos-boilerplate-ingress
  namespace: kratos-boilerplate
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
spec:
  tls:
  - hosts:
    - your-domain.com
    secretName: tls-secret
  rules:
  - host: your-domain.com
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: kratos-boilerplate-service
            port:
              number: 80
```

### 数据库部署

```yaml
# postgres.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: kratos-boilerplate
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:13
        env:
        - name: POSTGRES_DB
          value: kratos_prod
        - name: POSTGRES_USER
          value: postgres
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          limits:
            memory: "2Gi"
            cpu: "1000m"
          requests:
            memory: "1Gi"
            cpu: "500m"
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 20Gi
```

## 配置管理

### 环境变量配置

```bash
# .env.example
# 应用配置
ENV=production
APP_NAME=kratos-boilerplate

# 数据库配置
DATABASE_URL=postgresql://username:password@host:5432/database?sslmode=require
POSTGRES_DB=kratos_prod
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your-secure-password

# Redis 配置
REDIS_URL=redis://username:password@host:6379/0
REDIS_PASSWORD=your-redis-password

# 认证配置
JWT_SECRET_KEY=your-very-secure-jwt-secret-key-at-least-32-characters-long

# 外部服务
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your-email@example.com
SMTP_PASSWORD=your-email-password

# 监控配置
PROMETHEUS_ADDR=:9090
JAEGER_ENDPOINT=http://jaeger:14268/api/traces
```

### 配置文件管理

1. **开发环境**: `configs/config.yaml`
2. **测试环境**: `configs/test.yaml`  
3. **生产环境**: `configs/production.yaml`

每个环境使用独立的配置文件，通过环境变量覆盖敏感信息。

## 数据库迁移

### 迁移工具安装

```bash
# 安装 migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 迁移命令

```bash
# 创建新迁移
migrate create -ext sql -dir migrations -seq add_users_table

# 应用迁移
migrate -path ./migrations -database $DATABASE_URL up

# 回滚迁移
migrate -path ./migrations -database $DATABASE_URL down 1

# 检查迁移状态
migrate -path ./migrations -database $DATABASE_URL version
```

### 生产环境迁移

```bash
# 1. 备份数据库
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d_%H%M%S).sql

# 2. 应用迁移
migrate -path ./migrations -database $DATABASE_URL up

# 3. 验证迁移
psql $DATABASE_URL -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
```

## 监控和日志

### 应用监控

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'kratos-boilerplate'
    static_configs:
      - targets: ['app:8000']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

### 日志聚合

```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /app/logs/*.log
  fields:
    service: kratos-boilerplate
    environment: production

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "kratos-boilerplate-%{+yyyy.MM.dd}"
```

### 健康检查

应用提供多个健康检查端点：

- `/health` - 总体健康状态
- `/health/live` - 存活检查
- `/health/ready` - 就绪检查
- `/metrics` - Prometheus 指标

## 故障排查

### 常见问题

1. **应用无法启动**
   - 检查配置文件
   - 验证环境变量
   - 查看应用日志

2. **数据库连接失败**
   - 验证数据库服务状态
   - 检查连接字符串
   - 确认网络连通性

3. **性能问题**
   - 查看监控指标
   - 分析慢查询日志
   - 检查资源使用情况

### 日志查看

```bash
# Docker 环境
docker-compose logs -f app

# Kubernetes 环境
kubectl logs -f deployment/kratos-boilerplate -n kratos-boilerplate

# 传统部署
tail -f /var/log/kratos-boilerplate/access.log
```

## 安全考虑

### SSL/TLS 配置

1. 使用有效的 SSL 证书
2. 配置安全的密码套件
3. 启用 HSTS
4. 禁用不安全的协议版本

### 网络安全

1. 使用防火墙限制访问
2. 配置 VPN 访问内部服务
3. 实施 DDoS 保护
4. 定期安全审计

### 数据保护

1. 加密敏感数据
2. 定期备份
3. 访问权限控制
4. 审计日志记录

## 备份和恢复

### 数据库备份

```bash
# 自动备份脚本
#!/bin/bash
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/kratos_backup_$DATE.sql"

pg_dump $DATABASE_URL > $BACKUP_FILE
gzip $BACKUP_FILE

# 保留最近30天的备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete
```

### 恢复流程

```bash
# 1. 停止应用
docker-compose stop app

# 2. 恢复数据库
gunzip backup_file.sql.gz
psql $DATABASE_URL < backup_file.sql

# 3. 重启应用
docker-compose start app
```

## 版本发布

### 滚动更新

```bash
# Kubernetes 滚动更新
kubectl set image deployment/kratos-boilerplate app=kratos-boilerplate:v1.2.0 -n kratos-boilerplate

# 检查更新状态
kubectl rollout status deployment/kratos-boilerplate -n kratos-boilerplate

# 回滚
kubectl rollout undo deployment/kratos-boilerplate -n kratos-boilerplate
```

### 蓝绿部署

```bash
# 部署新版本到绿色环境
kubectl apply -f deployment-green.yaml

# 验证绿色环境
kubectl port-forward service/kratos-boilerplate-green 8080:80

# 切换流量
kubectl patch service kratos-boilerplate-service -p '{"spec":{"selector":{"version":"green"}}}'

# 清理蓝色环境
kubectl delete -f deployment-blue.yaml
```

## 性能优化

### 应用层优化

1. 启用 HTTP/2
2. 配置缓存策略
3. 优化数据库查询
4. 使用连接池

### 基础设施优化

1. 使用 CDN
2. 配置负载均衡
3. 启用压缩
4. 优化资源限制

## 总结

本部署指南提供了从开发到生产的完整部署方案。根据实际需求选择合适的部署方式：

- **开发环境**: 本地部署，快速迭代
- **测试环境**: Docker Compose，CI/CD 集成
- **生产环境**: Kubernetes，高可用性和可扩展性

确保在部署前仔细阅读每个步骤，并根据实际环境调整配置参数。