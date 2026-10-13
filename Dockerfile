# One image, several roles: the container command picks `api`, `worker` or `migrate`.
#   docker build -t netopology-backend .
#   docker run netopology-backend api

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=0.1.0-dev
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X github.com/Whitedragon115/Netopology-Backend/internal/server.Version=${VERSION}" \
      -o /out/netopology ./cmd/netopology

# distroless: no shell, no package manager, runs as uid 65532
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/netopology /usr/local/bin/netopology
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/netopology"]
CMD ["api"]
