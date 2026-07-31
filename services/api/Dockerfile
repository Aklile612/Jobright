# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /bin/api /app/api
RUN mkdir -p /app/storage/resumes
ENV UPLOAD_DIR=/app/storage/resumes
EXPOSE 8080
CMD ["/app/api"]
