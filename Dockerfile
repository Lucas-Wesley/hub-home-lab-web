FROM node:24-alpine AS web
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=web /src/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" -o /app .

# O Claude Code só mede o uso das contas do Claude (H11).
FROM node:24-alpine
RUN npm install -g @anthropic-ai/claude-code && npm cache clean --force
COPY --from=api /app /app
USER 1000:1000
ENV DATA_DIR=/data \
    FUSO=America/Sao_Paulo \
    DISABLE_AUTOUPDATER=1
ENTRYPOINT ["/app"]
