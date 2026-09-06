FROM docker.io/library/golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -mod=mod -trimpath -ldflags="-s -w" -o /out/muchi-api ./cmd/muchi-api

FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /app
COPY --from=build /out/muchi-api /app/muchi-api
COPY config /app/config
USER nonroot:nonroot
ENTRYPOINT ["/app/muchi-api"]
