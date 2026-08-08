# EthioDeploy / Nixpacks / Docker — build the Go API from this monorepo.
# Root has no go.mod, so Nixpacks alone fails; this Dockerfile is the build plan.

# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    python3 py3-pip \
    poppler-utils \
    fontconfig ttf-dejavu
WORKDIR /app

# PDF tailor script + fonts (optional; falls back gracefully if missing at runtime)
COPY services/api/scripts/ /app/scripts/
COPY --from=build /bin/api /app/api

RUN mkdir -p /app/storage/resumes /app/scripts/fonts \
  && pip3 install --break-system-packages --no-cache-dir pypdf reportlab pymupdf \
  || true

ENV UPLOAD_DIR=/app/storage/resumes
ENV PDF_PYTHON=python3
ENV STAMP_PDF_SCRIPT=/app/scripts/stamp_pdf.py
ENV PORT=8080

EXPOSE 8080
CMD ["/app/api"]
