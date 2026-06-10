# Manga Sources (MangaFire + MangaKakalot) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add MangaFire and MangaKakalot as new manga source adapters using Colly HTML scraping.

**Architecture:** Both adapters implement the existing `SourceAdapter` interface using Colly for HTML scraping with 1 req/sec rate limiting. No new dependencies.

**Tech Stack:** Go, Colly (already a dependency)

---

### Task 1: MangaFire Adapter

**Files:**
- Create: `scraper/internal/adapter/mangafire.go`
- Test: `scraper/internal/adapter/mangafire_test.go` (optional, live site dependant)
- Modify: `scraper/cmd/scraper/main.go` (add to sources)

- [ ] **Step 1: Create mangafire.go adapter**

Create `scraper/internal/adapter/mangafire.go`:

```go
package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/gocolly/colly/v2"
)

const mangafireBase = "https://mangafire.to"

type MangaFireAdapter struct {
	client *colly.Collector
	log    *slog.Logger
}

func NewMangaFireAdapter() *MangaFireAdapter {
	c := colly.NewCollector(
		colly.AllowedDomains("mangafire.to"),
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*mangafire.to*",
		Parallelism: 1,
		Delay:       1 * time.Second,
	})
	c.SetRequestTimeout(30 * time.Second)

	return &MangaFireAdapter{
		client: c,
		log:    slog.Default().With("adapter", "mangafire"),
	}
}

func (m *MangaFireAdapter) Name() string {
	return "mangafire"
}

func (m *MangaFireAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	pageURL := mangafireBase + "/home"
	var series []model.Series
	var mu sync.Mutex

	m.client.OnHTML(".original.card-lg .unit", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		link := e.ChildAttr("a.poster", "href")
		if link == "" {
			return
		}
		slug := strings.TrimPrefix(link, "/manga/")
		slug = strings.TrimSuffix(slug, "/")
		if slug == "" {
			return
		}

		title := e.ChildText(".info a")
		title = strings.TrimSpace(title)
		if title == "" {
			return
		}

		img := e.ChildAttr("a.poster img", "src")
		coverURL := ""
		if img != "" {
			coverURL = img
		}

		series = append(series, model.Series{
			SourceID: slug,
			SourceURL: mangafireBase + "/manga/" + slug,
			Title:    title,
			CoverURL: coverURL,
			Status:   "ONGOING",
			IsActive: true,
		})
	})

	if err := m.client.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangafire: visit %s: %w", pageURL, err)
	}
	m.client.Wait()

	return series, nil
}

func (m *MangaFireAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	pageURL := mangafireBase + "/manga/" + seriesID
	var chapters []model.Chapter
	var mu sync.Mutex

	m.client.OnHTML(".original.card-lg .unit .content[data-name=chap] li", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		link := e.ChildAttr("a", "href")
		if link == "" || !strings.Contains(link, "/en/chapter-") {
			return
		}

		text := e.ChildText("span:first-child")
		text = strings.TrimSpace(text)

		// Parse "Chap 23" or "Chap 13.2" etc
		numStr := strings.TrimPrefix(text, "Chap ")
		numStr = strings.TrimSpace(numStr)
		chapterNum, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return
		}

		titleText := e.ChildText("span b")
		var chapterTitle string
		if titleText != "" && titleText != "EN" {
			chapterTitle = titleText
		}

		chapterURL := mangafireBase + link

		chapters = append(chapters, model.Chapter{
			Number: chapterNum,
			Title:  chapterTitle,
			URL:    chapterURL,
			IsNew:  true,
		})
	})

	if err := m.client.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangafire: visit %s: %w", pageURL, err)
	}
	m.client.Wait()

	return chapters, nil
}
```

- [ ] **Step 2: Build to check compilation**

```bash
cd /Volumes/Seagate/developer/personal/manga-cdc/scraper
go build ./...
```

Expected: Success

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Seagate/developer/personal/manga-cdc
git add -f docs/superpowers/plans/2026-06-10-manga-sources-plan.md scraper/internal/adapter/mangafire.go
git commit -m "feat: add MangaFire HTML scraper adapter"
```

---

### Task 2: MangaKakalot Adapter

**Files:**
- Create: `scraper/internal/adapter/mangakakalot.go`
- Modify: `scraper/cmd/scraper/main.go` (add to sources)

- [ ] **Step 1: Create mangakakalot.go adapter**

Create `scraper/internal/adapter/mangakakalot.go`:

```go
package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/gocolly/colly/v2"
)

const mangakakalotBase = "https://www.mangakakalot.gg"

type MangaKakalotAdapter struct {
	client *colly.Collector
	log    *slog.Logger
}

func NewMangaKakalotAdapter() *MangaKakalotAdapter {
	c := colly.NewCollector(
		colly.AllowedDomains("www.mangakakalot.gg", "mangakakalot.gg"),
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*mangakakalot.gg*",
		Parallelism: 1,
		Delay:       1 * time.Second,
	})
	c.SetRequestTimeout(30 * time.Second)

	return &MangaKakalotAdapter{
		client: c,
		log:    slog.Default().With("adapter", "mangakakalot"),
	}
}

func (m *MangaKakalotAdapter) Name() string {
	return "mangakakalot"
}

func (m *MangaKakalotAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	pageURL := mangakakalotBase + "/manga-list?type=latest"
	var series []model.Series
	var mu sync.Mutex

	m.client.OnHTML(".manga-list .manga-item, .list-body .item", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		link := e.ChildAttr("a", "href")
		if link == "" {
			return
		}
		slug := strings.TrimPrefix(link, "/manga/")
		slug = strings.TrimSuffix(slug, "/")
		if slug == "" {
			return
		}

		title := e.ChildText(".item-title, .manga-name")
		title = strings.TrimSpace(title)
		if title == "" {
			return
		}

		img := e.ChildAttr("img", "src")
		coverURL := ""
		if img != "" {
			coverURL = img
		}

		series = append(series, model.Series{
			SourceID: slug,
			SourceURL: mangakakalotBase + "/manga/" + slug,
			Title:    title,
			CoverURL: coverURL,
			Status:   "ONGOING",
			IsActive: true,
		})
	})

	if err := m.client.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangakakalot: visit %s: %w", pageURL, err)
	}
	m.client.Wait()

	return series, nil
}

func (m *MangaKakalotAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	pageURL := mangakakalotBase + "/manga/" + seriesID
	var chapters []model.Chapter
	var mu sync.Mutex

	m.client.OnHTML(".chapter-list .row, .list-chapter li, table.chapter-table tr", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		link := e.ChildAttr("a", "href")
		if link == "" {
			return
		}

		numText := e.ChildText(".chapter-number, .chapter-name, span:first-child")
		numText = strings.TrimSpace(numText)
		numText = strings.TrimPrefix(numText, "Chapter ")
		numText = strings.TrimPrefix(numText, "Ch.")
		numText = strings.TrimSpace(numText)

		chapterNum, err := strconv.ParseFloat(numText, 64)
		if err != nil {
			return
		}

		titleText := e.ChildText(".chapter-name, .chapter-title")
		titleText = strings.TrimSpace(titleText)
		titleText = strings.TrimPrefix(titleText, "Chapter "+numText+" ")
		titleText = strings.TrimPrefix(titleText, "Ch. "+numText+" ")
		titleText = strings.TrimSpace(titleText)

		chapterURL := link
		if !strings.HasPrefix(link, "http") {
			chapterURL = mangakakalotBase + link
		}

		chapters = append(chapters, model.Chapter{
			Number: chapterNum,
			Title:  titleText,
			URL:    chapterURL,
			IsNew:  true,
		})
	})

	if err := m.client.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangakakalot: visit %s: %w", pageURL, err)
	}
	m.client.Wait()

	return chapters, nil
}
```

- [ ] **Step 2: Build to check compilation**

```bash
cd /Volumes/Seagate/developer/personal/manga-cdc/scraper
go build ./...
```

Expected: Success

- [ ] **Step 3: Register in main.go**

Modify `scraper/cmd/scraper/main.go` to add both adapters to the sources list:

```go
sources := []adapter.SourceAdapter{
    adapter.NewMangaDexAdapter(),
    adapter.NewMangaFireAdapter(),
    adapter.NewMangaKakalotAdapter(),
    // MangaPlus needs protobuf support — see mangaplus.go
    // adapter.NewMangaPlusAdapter(),
}
```

- [ ] **Step 4: Build and verify**

```bash
cd /Volumes/Seagate/developer/personal/manga-cdc/scraper
go vet ./...
go build ./...
```

Expected: No errors

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Seagate/developer/personal/manga-cdc
git add scraper/internal/adapter/mangakakalot.go scraper/cmd/scraper/main.go
git commit -m "feat: add MangaKakalot HTML scraper adapter"
```

---

### Task 3: E2E Verification

- [ ] **Step 1: Start the stack and run scraper**

```bash
cd /Volumes/Seagate/developer/personal/manga-cdc
docker compose up -d postgres redpanda connect
cd scraper
DATABASE_URL="postgres://mangacdc:mangacdc@localhost:5432/mangacdc?sslmode=disable" \
SCRAPE_INTERVAL_SECONDS="10" \
LOG_LEVEL="debug" \
go run ./cmd/scraper &
sleep 30
kill %1
```

- [ ] **Step 2: Verify series from both sources exist**

```bash
docker compose exec postgres psql -U mangacdc -d mangacdc -c \
  "SELECT source_id, title FROM manga_series WHERE source_id LIKE '%.%' OR source_id LIKE '%-%' LIMIT 10;"
```

Expected: Series from mangafire (slugs with dots like `one-piecee.dkw`) and mangakakalot (slugs with dashes like `one-piece-`)

- [ ] **Step 3: Push all commits**

```bash
cd /Volumes/Seagate/developer/personal/manga-cdc
git push
```
