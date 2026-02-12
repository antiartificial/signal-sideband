FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o signal-sideband .

FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm build

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/signal-sideband .
COPY --from=web /app/web/dist ./web/dist
COPY migrations/ ./migrations/

EXPOSE 3001
CMD ["./signal-sideband"]
