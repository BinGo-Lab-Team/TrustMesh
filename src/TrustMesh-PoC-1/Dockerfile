FROM golang:1.25 AS builder
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -o trustmesh .


FROM scratch
WORKDIR /app

COPY --from=builder /src/trustmesh .

ENTRYPOINT ["./trustmesh"]
