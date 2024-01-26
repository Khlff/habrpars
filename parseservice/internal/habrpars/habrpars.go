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
	"sync"
	"sync/atomic"
	"time"
)

const BaseHabrURL string = "https://habr.com"

type Parser struct {
	serviceDB  service.Service
	httpClient *http.Client
}

func NewParser(serv service.Service, httpClient *http.Client) Parser {
	return Parser{serviceDB: serv, httpClient: httpClient}
}

func (p *Parser) Start(ctx context.Context, workersNumber int) error {
	hubs, err := p.serviceDB.GetHubs(ctx)
	if err != nil {
		return err
	}
	log.Debug().Int("hubs collected", len(hubs)).Msg("")

	var wg sync.WaitGroup
	wg.Add(len(hubs))
	for _, hub := range hubs {
		func(hub service.Hub) {
			defer wg.Done()
			go func() {
				err = p.process(ctx, workersNumber, hub)
				if err != nil {
					log.Err(err).Str("hub id", hub.ID).Msg("error in process")
				}
			}()
		}(hub)
	}

	<-ctx.Done()
	wg.Wait()
	return ctx.Err()

}

func (p *Parser) process(ctx context.Context, workersNum int, hub service.Hub) error {
	processFunc := func() error {
		start := time.Now()
		articles, err := p.getArticlesFromHub(ctx, hub, workersNum)
		if err != nil {
			log.Err(err).Msg("")
			return err
		}
		log.Debug().Str("hub url", hub.URL).Msg("articles collected")

		successfullySaved, err := p.saveArticles(ctx, workersNum, articles)
		if err != nil {
			log.Err(err).Msg("")
			return err
		}
		log.Debug().
			Str("hub url", hub.URL).
			Int64("count of successfully saved articles", successfullySaved).
			Int64("dur ms", time.Since(start).Milliseconds()).
			Msg("")

		return nil
	}

	err := processFunc()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(hub.Timeout) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err = processFunc(); err != nil {
				return err
			}
		}
	}
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
	res, err := p.httpClient.Get(url)
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
	res, err := p.httpClient.Get(url)
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
