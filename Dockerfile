FROM golang:1.27-alpine AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o /eliava ./cmd/api

FROM alpine:3.20
RUN adduser -D app && mkdir -p /app/uploads && chown app /app/uploads
WORKDIR /app
COPY --from=build /eliava /app/eliava
COPY web/ /app/web/
USER app
ENV WEB_DIR=/app/web UPLOAD_DIR=/app/uploads PORT=8080
EXPOSE 8080
CMD ["/app/eliava"]
