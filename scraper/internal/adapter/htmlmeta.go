package adapter

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly/v2"
)

type openGraphMeta struct {
	Title       string
	Description string
	Image       string
}

func configureHTMLCollector(c *colly.Collector) {
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "manga-cdc-scraper/1.0")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml")
	})
}

func scrapeOpenGraph(c *colly.Collector, pageURL string) (openGraphMeta, error) {
	var meta openGraphMeta

	c.OnHTML("meta[property='og:title']", func(e *colly.HTMLElement) {
		if meta.Title == "" {
			meta.Title = strings.TrimSpace(e.Attr("content"))
		}
	})
	c.OnHTML("meta[property='og:description']", func(e *colly.HTMLElement) {
		if meta.Description == "" {
			meta.Description = strings.TrimSpace(e.Attr("content"))
		}
	})
	c.OnHTML("meta[property='og:image']", func(e *colly.HTMLElement) {
		if meta.Image == "" {
			meta.Image = strings.TrimSpace(e.Attr("content"))
		}
	})
	c.OnHTML("h1", func(e *colly.HTMLElement) {
		if meta.Title == "" {
			meta.Title = strings.TrimSpace(e.Text)
		}
	})

	if err := c.Visit(pageURL); err != nil {
		return meta, fmt.Errorf("visit %s: %w", pageURL, err)
	}
	c.Wait()
	return meta, nil
}

func normalizeHTMLStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ongoing", "publishing", "serializing", "active":
		return "ONGOING"
	case "completed", "complete", "finished":
		return "COMPLETED"
	case "hiatus", "paused":
		return "HIATUS"
	case "cancelled", "canceled", "dropped":
		return "CANCELLED"
	default:
		return "ONGOING"
	}
}
