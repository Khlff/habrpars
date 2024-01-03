package main

import (
	"context"
	"github.com/Khlff/habrpars/internal/habrpars"
	"github.com/Khlff/habrpars/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// DATABASE_URL_EXAMPLE := "postgres://username:password@localhost:5432/database_name"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	defer pool.Close()

	err = pool.Ping(ctx)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	dbService := service.Postgres{Pool: pool}
	parser := habrpars.NewParser(&dbService)

	log.Printf("Parser started")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)
	go func(interval int64, workersNumber int) {
		defer wg.Done()
		err := parser.Start(ctx, interval, workersNumber)
		if err != nil {
			log.Error().Err(err).Msg("")
			return
		}
	}(600, 5)

	<-sigChan
	cancel()
	wg.Wait()
}
