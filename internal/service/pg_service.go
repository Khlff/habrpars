package service

import (
	"context"
	"fmt"
	"github.com/Khlff/habrpars/internal/habrpars"
	"github.com/jackc/pgx/v5"
	"log"
)

type Service interface {
	GetHubs(ctx context.Context) ([]string, error)
	AddArticle(ctx context.Context, article habrpars.Article) error
}

type Postgres struct {
	DB *pgx.Conn
}

func (pg *Postgres) GetHubs(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf("SELECT 'hub_url' FROM 'hubs'")
	rows, err := pg.DB.Query(ctx, query)
	if err != nil {
		log.Printf("Error while exec sql querry: %v", err)
		return nil, err
	}
	defer rows.Close()

	var hubsLinks []string
	for rows.Next() {
		var hub string
		err = rows.Scan(&hub)
		if err != nil {
			return nil, err
		}
		hubsLinks = append(hubsLinks, hub)
	}

	return hubsLinks, nil
}

func (pg *Postgres) AddArticle(ctx context.Context, article habrpars.Article) error {
	return nil
}
