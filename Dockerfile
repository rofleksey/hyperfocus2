FROM node:24-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X 'hyperfocus/internal/container.Version=${VERSION}'" -o /out/hyperfocus ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /out/hyperfocus /usr/local/bin/hyperfocus
EXPOSE 8080
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8080/api/healthz || exit 1
ENTRYPOINT ["hyperfocus"]
