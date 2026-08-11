FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X github.com/beyondxinxin/nixvis/internal/util.Version=${VERSION} -X github.com/beyondxinxin/nixvis/internal/util.BuildTime=${BUILD_TIME} -X github.com/beyondxinxin/nixvis/internal/util.GitCommit=${GIT_COMMIT}" -o /out/nixvis ./cmd/nixvis/main.go

FROM alpine:3.23

WORKDIR /app
COPY --from=builder /out/nixvis /app/nixvis

VOLUME ["/app/nixvis_data"]
EXPOSE 8088

ENTRYPOINT ["/app/nixvis"]
