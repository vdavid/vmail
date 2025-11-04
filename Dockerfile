# --- Stage 1: Build the React front end ---
FROM node:25-alpine AS builder-fe

# Install pnpm
RUN npm install -g pnpm

WORKDIR /app/frontend

# Copy package manifests and install dependencies
# This layer is cached if the manifests don't change
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy the rest of the source and build the app
COPY frontend/ ./
# This creates the static files in /app/frontend/dist (or /build)
RUN pnpm build

# --- Stage 2: Build the Go back end ---
FROM golang:1.25-alpine AS builder-be

WORKDIR /app/backend

# Install build dependencies
RUN apk add --no-cache gcc g++

# Copy manifests and download modules
# This layer is cached if the manifests don't change
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the rest of the source code and build the binary
COPY backend/ ./
# Build a static binary for a minimal final image
RUN CGO_ENABLED=0 GOOS=linux go build -o /backend-server ./cmd/server/main.go

# --- Stage 3: Final image ---
# Use a minimal, secure base image
FROM alpine:3.22

# Install ca-certificates (for making TLS/SSL calls to IMAP)
RUN apk add --no-cache ca-certificates

# Set a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

# Copy the built Go binary from the 'builder-be' stage
COPY --from=builder-be /backend-server .

# Copy the built React app from the 'builder-fe' stage
# Assuming the build output is in a 'dist' folder
COPY --from=builder-fe /app/frontend/dist ./static

# Expose the port your Go app listens on
EXPOSE 8080

# The command to run the application
# The Go app serves files from './static'
CMD ["/backend-server"]