package service

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Service interface {
	GetHubs(ctx context.Context) ([]Hub, error)
	AddArticle(ctx context.Context, article Article) error
}

type Postgres struct {
	Pool *pgxpool.Pool
}

type Article struct {
	Header     string
	Date       time.Time
	URL        string
	AuthorName string
	AuthorLink string
	HubID      string
	Text       string
}

type Hub struct {
	ID  string
	URL string
}

type PgErrorCode string

var AlreadyExistError PgErrorCode = "23505"

func (pg *Postgres) GetHubs(ctx context.Context) ([]Hub, error) {
	conn, err := pg.Pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to acquire a connection: %v", err)
	}
	defer conn.Release()

	query := `SELECT id, url FROM hubs`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hubs []Hub
	for rows.Next() {
		var hub Hub
		err = rows.Scan(&hub.ID, &hub.URL)
		if err != nil {
			return nil, err
		}
		hubs = append(hubs, hub)
	}

	return hubs, nil
}

func (pg *Postgres) AddArticle(ctx context.Context, article Article) error {
	conn, err := pg.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("unable to acquire a connection: %v", err)
	}
	defer conn.Release()

	query := `INSERT INTO articles (
                        header, publication_date, url, text, author_name, author_url, hub_id
                        ) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = conn.Exec(
		ctx, query,
		article.Header,
		article.Date,
		article.URL,
		article.Text,
		article.AuthorName,
		article.AuthorLink,
		article.HubID,
	)
	if err != nil {
		return err
	}

	return nil
}
