package habrpars

import (
	"context"
	"errors"
	"fmt"
	"github.com/Khlff/habrpars/internal/service"
	"github.com/PuerkitoBio/goquery"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

const BaseHabrURL string = "https://habr.com"

type Parser struct {
	serviceDB service.Service
}

func NewParser(serv service.Service) Parser {
	return Parser{serviceDB: serv}
}

func (p *Parser) Start(ctx context.Context, intervalInSeconds int64, workersNumber int) error {
	err := p.serviceDB.CreateTables(ctx) // create tables if don`t exist
	if err != nil {
		return err
	}

	err = p.serviceDB.AddTestHubs(ctx) // add test hubs. delete later
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(intervalInSeconds) * time.Second)
	defer ticker.Stop()

	processFunc := func() error {
		start := time.Now()
		err := p.process(ctx, workersNumber)
		if err != nil {
			return err
		}
		log.Info().Int64("dur ms", time.Since(start).Milliseconds()).Msg("successful work")
		return nil
	}

	if err := processFunc(); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := processFunc(); err != nil {
				return err
			}
		}
	}
}

func (p *Parser) process(ctx context.Context, workersNum int) error {
	hubs, err := p.serviceDB.GetHubs(ctx)
	if err != nil {
		return err
	}
	log.Debug().Int("hubs collected", len(hubs)).Msg("")

	for _, hub := range hubs {
		log.Debug().Str("hub id", hub.ID).Msg("")
		articles, err := p.getArticlesFromHub(ctx, hub, workersNum)
		if err != nil {
			log.Err(err).Msg("")
			continue
		}
		log.Debug().Msg("articles collected")

		successfullySaved, err := p.saveArticles(ctx, workersNum, articles)
		if err != nil {
			log.Err(err).Msg("")
			continue
		}
		log.Debug().Int64("articles successfully saved", successfullySaved).Msg("")
	}
	return nil
}

func (p *Parser) getArticlesFromHub(ctx context.Context, hub service.Hub, workersNumber int) ([]service.Article, error) {
	links, err := p.getArticlesUrls(BaseHabrURL + hub.URL) // unsec
	if err != nil {
		return nil, err
	}

	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(workersNumber)
	articles := make([]service.Article, len(links))

	for i, link := range links {
		i, link := i, link
		g.Go(func() error {
			article, gErr := p.getArticle(BaseHabrURL + link)
			if gErr == nil {
				article.HubID = hub.ID
				articles[i] = article
			}
			return gErr
		})
	}

	if err = g.Wait(); err != nil {
		return nil, err
	}

	return articles, nil
}

func (p *Parser) getArticlesUrls(url string) ([]string, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Err(err).Msg("error while close body")
		}
	}(res.Body)

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	return getUrls(doc), nil
}

func (p *Parser) getArticle(url string) (service.Article, error) {
	res, err := http.Get(url)
	if err != nil {
		return service.Article{}, err
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Err(err).Msg("error while close body")
		}
	}(res.Body)

	if res.StatusCode != 200 {
		return service.Article{}, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return service.Article{}, err
	}

	article := service.Article{URL: url}

	author := getAuthor(doc)
	article.AuthorLink = author.URL
	article.AuthorName = author.Username

	article.Header = getHeader(doc)
	article.Date = getDate(doc)
	article.Text = getText(doc)

	return article, nil
}

func (p *Parser) saveArticles(ctx context.Context, workersNumber int, articles []service.Article) (int64, error) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workersNumber)

	var successfulSavedCounter atomic.Int64

	for _, article := range articles {
		article := article
		g.Go(func() error {
			err := p.serviceDB.AddArticle(ctx, article)
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) {
					if pgErr.Code == string(service.AlreadyExistError) {
						return nil
					}
				}
				return err
			}
			successfulSavedCounter.Add(1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return 0, err
	}

	return successfulSavedCounter.Load(), nil
}
