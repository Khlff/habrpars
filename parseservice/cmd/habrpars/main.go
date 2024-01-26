package main

import (
	"context"
	"github.com/Khlff/habrpars/internal/habrpars"
	"github.com/Khlff/habrpars/internal/httpserver"
	"github.com/Khlff/habrpars/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const MaxWorkersOnHub = 5

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
	httpClient := http.Client{Timeout: 5 * time.Second}
	parser := habrpars.NewParser(
		&dbService,
		&httpClient,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		httpserver.StartHTTPServer(ctx, &parser, cancel, MaxWorkersOnHub, os.Getenv("SERVER_ADDR"))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := parser.Start(ctx, MaxWorkersOnHub)
		if err != nil {
			log.Error().Err(err).Msg("")
			return
		}
	}()
	log.Printf("Parser started")

	<-sigChan
	cancel()
	wg.Wait()
}
