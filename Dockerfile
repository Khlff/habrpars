FROM golang:1.21 as builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o habrpars ./cmd/habrpars/main.go

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/habrpars /habrpars
ENTRYPOINT ["/habrpars"]