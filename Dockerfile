# syntax=docker/dockerfile:1.7
FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/rosetta-server ./cmd/rosetta-server

FROM scratch
COPY --from=build /out/rosetta-server /rosetta-server
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/rosetta-server"]
