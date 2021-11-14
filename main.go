package main

import (
	"context"
	"encoding/csv"
	"flag"
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
	"gopkg.in/yaml.v3"
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
	IncreaseMaxDepth(delta uint64)
}

type crawler struct {
	r        Requester
	res      chan CrawlResult
	visited  map[string]struct{}
	mu       sync.RWMutex
	maxDepth uint64
}

func NewCrawler(r Requester, maxDepth uint64) *crawler {
	return &crawler{
		r:        r,
		res:      make(chan CrawlResult),
		visited:  make(map[string]struct{}),
		mu:       sync.RWMutex{},
		maxDepth: maxDepth,
	}
}

func (c *crawler) IncreaseMaxDepth(delta uint64) {
	atomic.AddUint64(&c.maxDepth, delta)
}

func (c *crawler) Scan(ctx context.Context, url string, depth uint64) {
	if depth > c.maxDepth { //Проверяем то, что есть запас по глубине
		return
	}
	c.mu.RLock()
	_, ok := c.visited[url] //Проверяем, что мы ещё не смотрели эту страницу
	c.mu.RUnlock()
	if ok {
		return
	}
	log.Printf("Processing %s (%d/%d)", url, depth, c.maxDepth)
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
	MaxDepth   uint64 `yaml:"max_depth"`
	MaxResults int    `yaml:"max_results"`
	MaxErrors  int    `yaml:"max_errors"`
	Url        string `yaml:"url"`
	Timeout    int    `yaml:"timeout"` //in seconds
	Delta      uint64 `yaml:"delta"`
	Output     string `yaml:"output"`
}

func main() {
	configPath := flag.String("config", "config.yaml", "the config path")
	flag.Parse()

	cfg := Config{}

	configFile, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf(`error when opening config in "%s": %v`, *configPath, err)
	}

	err = yaml.Unmarshal(configFile, &cfg)
	if err != nil {
		log.Fatalf("error when reading config: %v", err)
	}

	var r Requester = NewRequester(time.Duration(cfg.Timeout) * time.Second)
	var cr Crawler = NewCrawler(r, cfg.MaxDepth)

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
			cr.IncreaseMaxDepth(cfg.Delta)
			log.Printf("Max depth increased")
		}
	}
}

func processResult(ctx context.Context, cancel func(), cr Crawler, cfg Config) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors

	outputFile := os.Stdout
	var err error

	if cfg.Output != "" {
		outputFile, err = os.Create(cfg.Output)
		if err != nil {
			log.Fatalf("error when openning csv-file: %v\n", err)
		}
	}
	defer outputFile.Close()
	csvWriter := csv.NewWriter(outputFile)

	// Формируем заголовок
	err = csvWriter.Write([]string{"URL", "Title"})
	if err != nil {
		log.Printf("error when writing csv header: %v\n", err)
		return
	}
	defer csvWriter.Flush()

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
					log.Println("Stopped because of max error limit exceeded")
					return
				}
			} else {
				maxResult--
				err := csvWriter.Write([]string{msg.Url, msg.Title})
				if err != nil {
					log.Println("Stopped because of error when ")
					return
				}

				// log.Printf("crawler result: [url: %s] Title: %s\n", msg.Url, msg.Title)
				if maxResult <= 0 {
					cancel()
					return
				}
			}
		}
	}
}
