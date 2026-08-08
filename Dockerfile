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

# The runtime base is glibc-based (Debian slim) rather than Alpine: onnxruntime
# and opencv-python-headless only publish manylinux (glibc) wheels, so a musl
# base cannot install them. The Go binary is built fully static (CGO_ENABLED=0)
# so it runs unchanged on glibc.
FROM python:3.12-slim
ENV ENVIRONMENT=production
ENV CGO_ENABLED=0
WORKDIR /opt

# Runtime dependencies for survivor-name OCR (RapidOCR / PaddleOCR-ONNX).
# libglib2.0-0 + libgomp1 are shared-lib deps of opencv/onnxruntime; libgl1 is a
# fallback for OpenCV. rapidocr_onnxruntime pulls the ONNX models via pip.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        curl ca-certificates libglib2.0-0 libgomp1 libgl1 && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd --system --gid 65532 hyperfocus && \
    useradd --system --uid 65532 --gid 65532 --home-dir /opt --shell /sbin/nologin hyperfocus && \
    pip install --no-cache-dir \
        rapidocr-onnxruntime==1.2.3 \
        onnxruntime==1.28.0 \
        opencv-python-headless==5.0.0.93 \
        numpy==2.5.1

COPY --from=build /out/hyperfocus /opt/hyperfocus
RUN chown -R hyperfocus:hyperfocus /opt
USER hyperfocus:hyperfocus
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/healthz || exit 1
CMD ["./hyperfocus"]
