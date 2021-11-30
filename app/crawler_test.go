package app

import (
	"context"
	"fmt"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

const (
	mainPageUrl = `https://telegram.org/help`
	mainPage    = `
		<html>
			<head><title>Telegram Title</title></head>
			<body>
				<a href="/first">First</a>
				<a href="/second">Second</a>
			</body>
		</html>`
	firstPageUrl = `https://telegram.org/first`
	firstPage    = `
		<html>
			<head><title>First Page Title</title></head>
			<body></body>
		</html>`
	secondPageUrl = `https://telegram.org/second`
	secondPage    = `
		<html>
			<head><title>Second Page Title</title></head>
			<body></body>
		</html>`
)


func TestCrawler(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(100)*time.Millisecond)
	mc := minimock.NewController(t)


	pages := map[string]string{
		mainPageUrl:   mainPage,
		firstPageUrl:  firstPage,
		secondPageUrl: secondPage,
	}

	requesterMock := NewRequesterMock(mc).GetMock.Set(
		func(ctx context.Context, url string) (Page, error) {
			p, found := pages[url]
			if !found {
				return nil, fmt.Errorf(`page not found for url "%s"`, url)
			}
			return NewPage(mainPageUrl, strings.NewReader(p))
		})

	cr := NewCrawler(requesterMock, 1)
	go cr.Scan(ctx, mainPageUrl, 0)

	expected := map[CrawlResult]struct{}{
		{Title: "Telegram Title", Url: mainPageUrl}:      {},
		{Title: "First Page Title", Url: firstPageUrl}:   {},
		{Title: "Second Page Title", Url: secondPageUrl}: {},
	}
	results := cr.ChanResult()
	visited := map[CrawlResult]struct{}{}

STOP:
	for {
		select {
		case <-ctx.Done():
			break STOP
		case result := <-results:
			visited[result] = struct{}{}
		}
	}

	assert.Equal(t, expected, visited)
}
