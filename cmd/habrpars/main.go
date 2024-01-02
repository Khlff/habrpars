package main

import (
	"context"
	"github.com/Khlff/habrpars/internal/habrpars"
	"github.com/Khlff/habrpars/internal/service"
	"github.com/jackc/pgx/v5"
	"log"
	"os"
)

// urlExample := "postgres://username:password@localhost:5432/database_name"

func main() {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	defer conn.Close(ctx)

	err = conn.Ping(ctx)
	if err != nil {
		log.Fatal(err)
	}

	dbService := service.Postgres{DB: conn}
	parser := habrpars.NewParser(&dbService)
	if err = parser.Start(ctx, 5, 5); err != nil {
		log.Fatal(err)
	}
}
