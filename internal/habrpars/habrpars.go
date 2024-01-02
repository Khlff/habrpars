package habrpars

import (
	"context"
	"fmt"
	"github.com/Khlff/habrpars/internal/service"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const BaseHabrUrl string = "https://habr.com"

type Article struct {
	Header     string
	Date       time.Time
	Link       string
	Author     string
	AuthorLink string
	Hub        string
}

type Parser struct {
	timeoutInSeconds int64
	serviceDB        service.Service
}

func NewParser(serv service.Service) Parser {
	return Parser{serviceDB: serv}
}

func (p *Parser) Start(ctx context.Context, timeout int64, workersNumber int) error {
	ticker := time.NewTicker(time.Duration(timeout) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := p.process(ctx, workersNumber)
			if err != nil {
				return err
			}
		}
	}
}

func (p *Parser) process(ctx context.Context, workersNum int) error {
	hubsUrls, err := p.serviceDB.GetHubs(ctx)
	if err != nil {
		return err
	}

	for _, hubUrl := range hubsUrls {
		articles, err := p.getArticlesFromHub(ctx, hubUrl, workersNum)
		if err != nil {
			log.Println(err)
			continue
		}

		err = p.saveArticles(ctx, workersNum, &articles)
		if err != nil {
			log.Println(err)
			continue
		}
	}
	return nil
}

func (p *Parser) getArticlesFromHub(ctx context.Context, hubUrl string, workersNumber int) ([]Article, error) {
	links, err := p.getLinks(BaseHabrUrl + hubUrl)
	if err != nil {
		return nil, err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workersNumber)
	articles := make([]Article, len(links))

	for i, link := range links {
		i, link := i, link
		g.Go(func() error {
			article, err := p.getArticle(BaseHabrUrl + link)
			if err == nil {
				article.Hub = hubUrl
				articles[i] = article
			}
			return err
		})
	}

	if err = g.Wait(); err != nil {
		return nil, err
	}

	return articles, nil
}

func (p *Parser) getLinks(url string) ([]string, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(res.Body)

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	var links []string

	doc.Find(".tm-article-snippet.tm-article-snippet > h2").Each(func(i int, s *goquery.Selection) {
		link, exist := s.Find("a").Attr("href")
		if exist {
			links = append(links, link)
		}
	})

	return links, nil
}

func (p *Parser) getArticle(url string) (Article, error) {
	res, err := http.Get(url)
	if err != nil {
		return Article{}, err
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(res.Body)

	if res.StatusCode != 200 {
		return Article{}, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return Article{}, err
	}

	article := Article{Link: url}

	articleData := doc.Find("div.tm-article-presenter__body > div.tm-misprint-area > div > article > div.tm-article-presenter__header")

	header := articleData.Find("div > h1")
	article.Header = header.Text()

	date := articleData.Find("div > div.tm-article-snippet__meta-container > div > span > span > span > time")
	datetime, exist := date.Attr("datetime")
	if exist {
		parsedTime, err := time.Parse(time.RFC3339, datetime)
		if err == nil {
			article.Date = parsedTime
		}
	}

	authorData := articleData.Find("div > div.tm-article-snippet__meta-container > div > span > span")
	authorName := authorData.Find("a").Text()
	article.Author = strings.TrimSpace(authorName)

	authorLink, exist := authorData.Find("a").Attr("href")
	if exist {
		article.AuthorLink = authorLink
	}

	return article, nil
}

func (p *Parser) saveArticles(ctx context.Context, workersNumber int, articles *[]Article) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workersNumber)

	for _, article := range *articles {
		article := article
		g.Go(func() error {
			err := p.serviceDB.AddArticle(ctx, article)
			if err != nil {
				log.Println(err)
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
