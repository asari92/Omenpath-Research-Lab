FROM node:22-alpine AS web-build

WORKDIR /src
COPY web/package.json web/package-lock.json ./web/
RUN npm ci --prefix web
COPY web ./web
COPY data ./data
COPY 01_AI_WORKLOG_CURRENT.md ./
RUN npm --prefix web run build

FROM golang:1.26.4-alpine AS go-build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY data ./data
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/omenpath ./cmd/server

FROM alpine:3.22

RUN addgroup -S -g 10001 omenpath && adduser -S -D -H -u 10001 -G omenpath omenpath
WORKDIR /app
COPY --from=go-build /out/omenpath /app/omenpath
COPY --from=web-build /src/web/dist /app/web
RUN mkdir -p /var/lib/omenpath && chown -R 10001:10001 /var/lib/omenpath

USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/app/omenpath"]
