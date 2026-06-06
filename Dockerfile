FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/api /bin/api
COPY ./web ./web
COPY ./migrations ./migrations


EXPOSE ${SERVER_PORT}

ENTRYPOINT ["/bin/api"]
