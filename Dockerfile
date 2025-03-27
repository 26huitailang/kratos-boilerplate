FROM openeuler/openeuler:24.03-lts AS builder

# 安装构建工具
RUN dnf update -y && dnf install -y \
    wget \
    tar \
    nodejs \
    npm \
    git \
    make \
    && dnf clean all \
    && rm -rf /var/cache/dnf/*

# 安装指定版本的 Go
RUN wget https://go.dev/dl/go1.24.1.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.24.1.linux-amd64.tar.gz && \
    rm go1.24.1.linux-amd64.tar.gz

# 设置 Go 环境
ENV GOPROXY=https://goproxy.cn
ENV GO111MODULE=on
ENV PATH=$PATH:/usr/local/go/bin
ENV GOROOT=/usr/local/go

# 复制源代码
COPY . /src
WORKDIR /src

# 构建后端
RUN cd /src && make build

# 构建前端
RUN cd /src/frontend && \
    npm install && \
    npm run build

# 最终镜像
FROM openeuler/openeuler:24.03-lts

RUN dnf update -y && dnf install -y \
    ca-certificates \
    net-tools \
    && dnf clean all \
    && rm -rf /var/cache/dnf/*

# 复制构建产物
COPY --from=builder /src/bin /app/bin
COPY --from=builder /src/frontend/dist /app/frontend

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["/app/bin/xredline", "-conf", "/data/conf"]
