# syntax=docker/dockerfile:1

FROM node:22-alpine AS web
ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ENV HTTP_PROXY=${HTTP_PROXY} HTTPS_PROXY=${HTTPS_PROXY}
WORKDIR /build/web
COPY web/package.json web/package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS backend
ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ENV HTTP_PROXY=${HTTP_PROXY} HTTPS_PROXY=${HTTPS_PROXY}
WORKDIR /build
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /build/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -tags 'embed migration' -trimpath -ldflags '-s -w' -o /paopao .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
ENV HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" GIN_MODE=release
WORKDIR /app
COPY --from=backend /paopao /app/paopao
RUN mkdir -p /app/custom/data
EXPOSE 8008
HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=12 \
  CMD wget -q -O - 'http://127.0.0.1:8008/v1/posts?style=newest' | /bin/busybox grep -Eq '"code"[[:space:]]*:[[:space:]]*0([^0-9]|$)' || exit 1
CMD ["/app/paopao", "serve"]
