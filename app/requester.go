package app

import (
	"context"
	"net/http"
	"time"
)

type requester struct {
	timeout time.Duration
}

func NewRequester(timeout time.Duration) requester {
	return requester{timeout: timeout}
}

// Можно протестировать, если вынести http.Client и http.NewRequest в отдельную структуру и
// закрыться от неё интерфейсом.
// Но в этом случае здесь почти не остаётся логики, кроме отмены контекста и оборачивания
// результата запроса в Page
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
