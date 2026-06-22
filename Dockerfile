# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/gochain ./cmd/gochain

# ---- run stage ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bin/gochain /gochain
EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/gochain"]
