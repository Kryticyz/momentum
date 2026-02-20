# Stage 1: Frontend build
FROM oven/bun:1-alpine AS frontend-builder
WORKDIR /build

# Copy frontend source and OpenAPI spec for type generation.
COPY apps/dashboard-frontend/package.json apps/dashboard-frontend/bun.lock* apps/dashboard-frontend/
COPY docs/openapi-dashboard-backend.yaml docs/

WORKDIR /build/apps/dashboard-frontend
RUN bun install --frozen-lockfile

# Copy remaining frontend source.
COPY apps/dashboard-frontend/ ./

# generate:types needs the spec at ../../docs/ (relative to frontend dir).
# Vite build outputs to ../../services/dashboard-backend/frontend/dist.
RUN bun run generate:types && bun run build

# Stage 2: Backend build
FROM golang:1.24-alpine AS backend-builder
WORKDIR /build

COPY services/dashboard-backend/go.mod services/dashboard-backend/go.sum ./
RUN go mod download

COPY services/dashboard-backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o momentum-dashboard .

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=backend-builder /build/momentum-dashboard .
COPY --from=frontend-builder /build/services/dashboard-backend/frontend/dist ./frontend/dist

EXPOSE 8080

ENTRYPOINT ["./momentum-dashboard"]
