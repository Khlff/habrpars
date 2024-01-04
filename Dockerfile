FROM golang:1.21 as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod tidy
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /habrpars ./cmd/habrpars/main.go

FROM scratch
COPY --from=builder /habrpars /habrpars
ENTRYPOINT ["/habrpars"]