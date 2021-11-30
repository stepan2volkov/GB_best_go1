package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	mainUrl = `https://telegram.org/help`
	pageContent    = `
		<html>
			<head><title>Telegram Title</title></head>
			<body>
				<a href="/first">First</a>
				<a href="/second">Second</a>
			</body>
		</html>`
	firstUrl = `https://telegram.org/first`
	secondUrl = `https://telegram.org/second`
)

func TestPage(t *testing.T) {
	doc := strings.NewReader(pageContent)
	testPage, err := NewPage(mainUrl, doc)
	assert.Equal(t, nil, err, "unexpected error when creating page")
	assert.Equal(t, "Telegram Title", testPage.GetTitle())
	assert.Equal(t, []string{firstUrl, secondUrl}, testPage.GetLinks())
}
