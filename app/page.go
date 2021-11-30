package app

import (
	"fmt"
	"io"
	"log"
	netURL "net/url"

	"github.com/PuerkitoBio/goquery"
)

// TODO: var _ app.Page = &page{}

type page struct {
	doc     *goquery.Document
	baseURL string
}

func NewPage(url string, raw io.Reader) (*page, error) {
	doc, err := goquery.NewDocumentFromReader(raw)
	if err != nil {
		return nil, err
	}
	// Для относительных ссылок
	parsedBaseURL, _ := netURL.Parse(url)
	baseURL := fmt.Sprintf("%s://%s", parsedBaseURL.Scheme, parsedBaseURL.Host)
	return &page{doc: doc, baseURL: baseURL}, nil
}

func (p *page) GetTitle() string {
	return p.doc.Find("title").First().Text()
}

func (p *page) GetLinks() []string {
	var urls []string
	p.doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, ok := s.Attr("href")
		if ok {
			newURL, err := p.makeLink(url)
			if err != nil {
				log.Printf(`crawler result: cannot generate link for: "%s" (err: "%s")\n`, url, err.Error())
				return
			}
			urls = append(urls, newURL)
		}
	})
	return urls
}

// makeLink creates absolute link if we've got a relative one
func (p *page) makeLink(url string) (string, error) {
	parsedURL, err := netURL.Parse(url)
	if err != nil {
		return "", err
	}

	// Для относительных ссылок добавляем хост страницы, с которой взяли ссылку
	if parsedURL.Host == "" {
		return p.baseURL + url, nil
	}

	// Добавляем схему, если она не указана. Для ссылок вида "//core.telegram.org/api"
	if parsedURL.Scheme == "" {
		return "https:" + url, nil
	}
	return url, nil
}
