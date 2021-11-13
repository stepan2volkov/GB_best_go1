package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	netURL "net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type CrawlResult struct {
	Err   error
	Title string
	Url   string
}

type Page interface {
	GetTitle() string
	GetLinks() []string
}

type page struct {
	doc     *goquery.Document
	baseURL string
}

func NewPage(url string, raw io.Reader) (Page, error) {
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
				log.Printf(`crawler result: cannot generate link for: %s (err: %s)\n`, url, err.Error())
				return
			}
			urls = append(urls, newURL)
		}
	})
	return urls
}

func (p *page) makeLink(url string) (string, error) {
	parsedURL, err := netURL.Parse(url)
	if err != nil {
		return "nil", err
	}

	if parsedURL.Host == "" {
		return p.baseURL + url, nil
	}

	newURL := parsedURL.Host + parsedURL.Path
	if parsedURL.RawQuery != "" {
		newURL = fmt.Sprintf("%s?%s", newURL, parsedURL.RawQuery)
	}
	if parsedURL.Scheme == "" {
		newURL = "https://" + newURL
	} else {
		newURL = fmt.Sprintf("%s://%s", parsedURL.Scheme, newURL)
	}

	return newURL, nil
}

type Requester interface {
	Get(ctx context.Context, url string) (Page, error)
}

type requester struct {
	timeout time.Duration
}

func NewRequester(timeout time.Duration) requester {
	return requester{timeout: timeout}
}

func (r requester) Get(ctx context.Context, url string) (Page, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		cl := &http.Client{
			Timeout: r.timeout,
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		body, err := cl.Do(req)
		if err != nil {
			return nil, err
		}
		defer body.Body.Close()
		page, err := NewPage(url, body.Body)
		if err != nil {
			return nil, err
		}
		return page, nil
	}
}

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth uint64)
	ChanResult() <-chan CrawlResult
}

type crawler struct {
	r       Requester
	res     chan CrawlResult
	visited map[string]struct{}
	mu      sync.RWMutex
	cfg     *Config
}

func NewCrawler(r Requester, cfg *Config) *crawler {
	return &crawler{
		r:       r,
		res:     make(chan CrawlResult),
		visited: make(map[string]struct{}),
		mu:      sync.RWMutex{},
		cfg:     cfg,
	}
}

func (c *crawler) Scan(ctx context.Context, url string, depth uint64) {
	if depth > c.cfg.MaxDepth { //Проверяем то, что есть запас по глубине
		return
	}
	c.mu.RLock()
	_, ok := c.visited[url] //Проверяем, что мы ещё не смотрели эту страницу
	c.mu.RUnlock()
	if ok {
		return
	}
	log.Printf("Checking %s (depth %d/%d)", url, depth, c.cfg.MaxDepth)
	select {
	case <-ctx.Done(): //Если контекст завершен - прекращаем выполнение
		return
	default:
		page, err := c.r.Get(ctx, url) //Запрашиваем страницу через Requester
		if err != nil {
			c.res <- CrawlResult{Err: err} //Записываем ошибку в канал
			return
		}
		c.mu.Lock()
		c.visited[url] = struct{}{} //Помечаем страницу просмотренной
		c.mu.Unlock()
		c.res <- CrawlResult{ //Отправляем результаты в канал
			Title: page.GetTitle(),
			Url:   url,
		}
		for _, link := range page.GetLinks() {
			go c.Scan(ctx, link, depth+1) //На все полученные ссылки запускаем новую рутину сборки
		}
	}
}

func (c *crawler) ChanResult() <-chan CrawlResult {
	return c.res
}

//Config - структура для конфигурации
type Config struct {
	MaxDepth   uint64
	MaxResults int
	MaxErrors  int
	Url        string
	Timeout    int //in seconds
}

func main() {

	cfg := Config{
		MaxDepth:   5,
		MaxResults: 10000,
		MaxErrors:  5000,
		Url:        "https://telegram.org",
		Timeout:    100,
	}

	var r Requester = NewRequester(time.Duration(cfg.Timeout) * time.Second)
	var cr Crawler = NewCrawler(r, &cfg)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	go cr.Scan(ctx, cfg.Url, 0)            //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg) //Обрабатываем результаты в отдельной рутине

	sigStopCh := make(chan os.Signal, 1)     //Создаем канал для приема сигнала завершения
	signal.Notify(sigStopCh, syscall.SIGINT) //Подписываемся на сигнал SIGINT

	sigCh := make(chan os.Signal, 1)      //Создаем канал для приема сигнала завершения
	signal.Notify(sigCh, syscall.SIGUSR1) //Подписываемся на сигнал SIGINT

	for {
		select {
		case <-ctx.Done(): //Если всё завершили - выходим
			return
		case <-sigStopCh:
			log.Println("Gracefull shutdown started") // Приятно знать, что мы действительно обработали сигнал
			cancel()                                  //Если пришёл сигнал SigInt - завершаем контекст
		case <-sigCh:
			log.Printf(`Max depth increased. Current value is "%d"\n`, cfg.MaxDepth)
			atomic.AddUint64(&cfg.MaxDepth, 2)
		}
	}
}

func processResult(ctx context.Context, cancel func(), cr Crawler, cfg Config) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cr.ChanResult():
			if msg.Err != nil {
				maxErrors--
				log.Printf("crawler result return err: %s\n", msg.Err.Error())
				if maxErrors <= 0 {
					cancel()
					log.Println("Stopped because of max error limit.")
					return
				}
			} else {
				maxResult--
				log.Printf("crawler result: [url: %s] Title: %s\n", msg.Url, msg.Title)
				if maxResult <= 0 {
					cancel()
					return
				}
			}
		}
	}
}
