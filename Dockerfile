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
RUN go build -o /out/hyperfocus ./cmd/server

FROM python:3.12-alpine
ENV ENVIRONMENT=production
ENV CGO_ENABLED=0
WORKDIR /opt

# Runtime dependencies for survivor-name OCR (RapidOCR / PaddleOCR-ONNX).
# opencv-python-headless needs libGL/libgomp; rapidocr_onnxruntime pulls ONNX
# models on first run via pip (a few MB) at install time.
RUN apk update && \
    apk add --no-cache curl ca-certificates libgomp mesa-gl && \
    update-ca-certificates && \
    addgroup -g 65532 -S hyperfocus && \
    adduser -S -u 65532 -G hyperfocus -h /opt -s /sbin/nologin hyperfocus && \
    pip install --no-cache-dir \
        rapidocr-onnxruntime==1.2.3 \
        onnxruntime==1.28.0 \
        opencv-python-headless==5.0.0.93 \
        numpy==2.5.1

COPY --from=build /out/hyperfocus /opt/hyperfocus
COPY scripts/ocr/ /opt/ocr/
RUN chown -R hyperfocus:hyperfocus /opt
USER hyperfocus:hyperfocus
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/healthz || exit 1
CMD ["./hyperfocus"]
