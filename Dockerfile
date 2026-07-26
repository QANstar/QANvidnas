# Stage 1: Build React frontend
FROM node:22-alpine AS client-builder
WORKDIR /client
COPY client/package*.json ./
RUN npm config set registry https://registry.npmmirror.com && npm ci
COPY client/ ./
RUN npm run build

# Stage 2: Build Go server with embedded frontend
FROM golang:1.22-alpine3.20 AS server-builder
WORKDIR /server

# Install build dependencies (required for CGO + SQLite)
RUN apk add --no-cache gcc musl-dev

ENV GOPROXY=https://goproxy.cn,direct

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
COPY --from=client-builder /client/dist ./internal/router/web/dist/

RUN CGO_ENABLED=1 go build -tags "sqlite_fts5" -o /qanvidnas .

# Stage 3: Production image
FROM golang:1.22-alpine3.20

# Runtime dependencies
RUN apk add --no-cache ffmpeg ca-certificates tzdata

WORKDIR /app
COPY --from=server-builder /qanvidnas .
RUN mkdir -p /app/data

EXPOSE 6666
ENV TZ=Asia/Shanghai

CMD ["./qanvidnas"]
