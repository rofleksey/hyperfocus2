FROM node:22-alpine AS frontend
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/dist ./web/dist
# Fully static binary (no CGO): OCR now lives in an external microservice, so
# the runtime no longer needs Python / onnxruntime / OpenCV.
RUN CGO_ENABLED=0 go build -o /out/hyperfocus ./cmd/server

# Minimal runtime: the Go binary is static, so Alpine (musl) runs it unchanged.
# ca-certificates is needed for outbound TLS (Twitch API); curl powers the
# healthcheck.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl && \
    addgroup -S -g 65532 hyperfocus && \
    adduser -S -D -H -u 65532 -G hyperfocus -h /opt hyperfocus
WORKDIR /opt
COPY --from=build /out/hyperfocus /opt/hyperfocus
USER hyperfocus:hyperfocus
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/healthz || exit 1
CMD ["./hyperfocus"]
