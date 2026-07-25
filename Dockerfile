# Stage 1: Build React frontend
FROM node:22-alpine AS client-builder
WORKDIR /client
COPY client/package*.json ./
RUN npm ci
COPY client/ ./
RUN npm run build

# Stage 2: Build Go server with embedded frontend
FROM golang:1.22-alpine AS server-builder
WORKDIR /server

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

COPY server/go.mod ./
COPY server/ ./
COPY --from=client-builder /client/dist ./internal/router/web/dist/

RUN go mod tidy
RUN CGO_ENABLED=1 go build -o /qanvidnas .

# Stage 3: Final image
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ffmpeg ca-certificates tzdata

WORKDIR /app

COPY --from=server-builder /qanvidnas .

# Create data directory
RUN mkdir -p /app/data

EXPOSE 6666

ENV TZ=Asia/Shanghai

VOLUME ["/app/data", "/app/config.yaml", "/media"]

CMD ["./qanvidnas"]
