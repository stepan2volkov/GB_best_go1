package app

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
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

type Requester interface {
	Get(ctx context.Context, url string) (Page, error)
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
