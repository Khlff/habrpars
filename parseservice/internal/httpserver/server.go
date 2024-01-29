package httpserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/Khlff/habrpars/internal/habrpars"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

func StartHTTPServer(
	ctx context.Context,
	parser *habrpars.Parser,
	cancel context.CancelFunc,
	maxWorkersOnHub int,
	addr string) {
	handler := http.NewServeMux()
	handler.HandleFunc("/restart", func(w http.ResponseWriter, r *http.Request) {
		log.Info().Msg("Received restart request")
		ctx, cancel = context.WithCancel(context.Background())
		go func() {
			err := parser.Start(ctx, maxWorkersOnHub)
			if err != nil {
				log.Error().Err(err).Msg("")
				return
			}
		}()
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info().Str("address", addr).Msg("HTTP server started")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("HTTP server error")
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Error shutting down server: %v\n", err)
	}
	log.Info().Msg("HTTP server was shutdown")
}
