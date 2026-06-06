FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO precisa estar habilitado por causa do mattn/go-sqlite3 (usado pelo whatsmeow).
# Link estatico com musl para o binario rodar na imagem alpine final sem depender
# de bibliotecas do sistema.
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "sqlite_omit_load_extension" \
    -ldflags '-linkmode external -extldflags "-static"' \
    -o /bin/api ./cmd/api/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/api /bin/api
COPY ./web ./web
COPY ./migrations ./migrations


EXPOSE ${SERVER_PORT}

ENTRYPOINT ["/bin/api"]
