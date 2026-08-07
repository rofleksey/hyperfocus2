FROM node:22-alpine AS frontend
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/dist ./web/dist
RUN go build -o /out/hyperfocus ./cmd/server

FROM alpine
ENV ENVIRONMENT=production
ENV CGO_ENABLED=0
WORKDIR /opt
RUN apk update && \
    apk add --no-cache curl ca-certificates && \
    update-ca-certificates && \
    addgroup -g 65532 -S hyperfocus && \
    adduser -S -u 65532 -G hyperfocus -h /opt -s /sbin/nologin hyperfocus
COPY --from=build /out/hyperfocus /opt/hyperfocus
RUN chown hyperfocus:hyperfocus /opt/hyperfocus
USER hyperfocus:hyperfocus
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/healthz || exit 1
CMD ["./hyperfocus"]
