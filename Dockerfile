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
    python3 py3-pip py3-virtualenv \
    poppler-utils \
    fontconfig ttf-dejavu bash curl
WORKDIR /app

COPY services/api/scripts/ /app/scripts/
COPY --from=build /bin/api /app/api

RUN mkdir -p /app/storage/resumes /app/scripts/fonts \
  && python3 -m venv /app/.venv-pdf \
  && /app/.venv-pdf/bin/pip install --no-cache-dir --upgrade pip \
  && /app/.venv-pdf/bin/pip install --no-cache-dir pypdf reportlab pymupdf

ENV PATH=/app/.venv-pdf/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
ENV UPLOAD_DIR=/app/storage/resumes
ENV PDF_PYTHON=/app/.venv-pdf/bin/python
ENV STAMP_PDF_SCRIPT=/app/scripts/stamp_pdf.py
ENV EXTRACT_PDF_SCRIPT=/app/scripts/extract_pdf_text.py
ENV PORT=8080

EXPOSE 8080
CMD ["/app/api"]
