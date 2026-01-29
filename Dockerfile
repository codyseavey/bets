# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=1 GOOS=linux go build -o server .

# Stage 3: Production
FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=backend-builder /app/server .
COPY --from=frontend-builder /frontend/dist ./frontend/dist
ENV PORT=8080
EXPOSE 8080
CMD ["./server"]
