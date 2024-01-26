package habrpars

import (
	"github.com/PuerkitoBio/goquery"
	"strings"
	"time"
)

const urlsSelector string = ".tm-article-snippet.tm-article-snippet > h2"
const headerSelector string = "div.tm-article-presenter__body > div.tm-misprint-area > div > article > div.tm-article-presenter__header > div > h1"
const dateSelector string = "div.tm-article-presenter__body > div.tm-misprint-area > div > article > div.tm-article-presenter__header > div > div.tm-article-snippet__meta-container > div > span > span > span > time"
const authorSelector string = "div.tm-article-presenter__body > div.tm-misprint-area > div > article > div.tm-article-presenter__header > div > div.tm-article-snippet__meta-container > div > span > span"
const textBodySelector string = "#post-content-body > div:nth-child(1) > div > div"
const textParSelector = textBodySelector + " > p"

type AuthorData struct {
	Username string
	URL      string
}

func getHeader(doc *goquery.Document) string {
	sel := doc.Find(headerSelector)
	return sel.Text()
}

func getDate(doc *goquery.Document) time.Time {
	sel := doc.Find(dateSelector)
	datetime, exist := sel.Attr("datetime")
	if exist {
		parsedTime, err := time.Parse(time.RFC3339, datetime)
		if err == nil {
			return parsedTime
		}
	}

	return time.Time{}
}

func getAuthor(doc *goquery.Document) AuthorData {
	authorInfo := doc.Find(authorSelector)
	authorName := authorInfo.Find("a").Text()

	var authorURL string
	hrefAuthorURL, exist := authorInfo.Find("a").Attr("href")
	if exist {
		authorURL = hrefAuthorURL
	}

	return AuthorData{
		Username: authorName,
		URL:      authorURL,
	}
}

func getText(doc *goquery.Document) string {
	var textBuilder strings.Builder

	par := doc.Find(textParSelector)
	par.Each(func(i int, s *goquery.Selection) {
		textBuilder.WriteString(s.Text())
		textBuilder.WriteString("\n")
	})

	if textBuilder.Len() != 0 {
		return textBuilder.String()
	}

	withoutParCase := doc.Find(textBodySelector)
	return strings.TrimSpace(withoutParCase.Text())
}

func getUrls(doc *goquery.Document) []string {
	var links []string

	doc.Find(urlsSelector).Each(func(i int, s *goquery.Selection) {
		link, exist := s.Find("a").Attr("href")
		if exist {
			links = append(links, link)
		}
	})
	return links
}
