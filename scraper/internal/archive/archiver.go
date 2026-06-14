package archive

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"golang.org/x/sync/errgroup"
)

const (
	maxArchivePages   = 100
	maxImageBytes     = 10 * 1024 * 1024
	downloadTimeout   = 60 * time.Second
)

type Archiver struct {
	BaseDir    string
	HTTPClient *http.Client
}

func NewArchiver(baseDir string) *Archiver {
	return &Archiver{
		BaseDir: baseDir,
		HTTPClient: &http.Client{
			Timeout: downloadTimeout,
		},
	}
}

// ArchiveChapter downloads all images from the provided urls and zips them into a .cbz file
func (a *Archiver) ArchiveChapter(ctx context.Context, series model.Series, chapter model.Chapter, imageUrls []string) error {
	if len(imageUrls) == 0 {
		return fmt.Errorf("no images provided for chapter archive")
	}
	if len(imageUrls) > maxArchivePages {
		return fmt.Errorf("chapter exceeds max archive pages (%d)", maxArchivePages)
	}

	// Create directory if it doesn't exist
	seriesDir := filepath.Join(a.BaseDir, sanitizePath(series.Title))
	if err := os.MkdirAll(seriesDir, 0755); err != nil {
		return fmt.Errorf("failed to create series directory: %w", err)
	}

	cbzFilename := fmt.Sprintf("%s - Ch. %v.cbz", series.Title, chapter.Number)
	cbzPath := filepath.Join(seriesDir, sanitizePath(cbzFilename))

	// Create a temporary file to stream zip contents into
	tmpPath := cbzPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer func() {
		file.Close()
		if err != nil {
			os.Remove(tmpPath) // clean up on failure
		}
	}()

	zipWriter := zip.NewWriter(file)

	type downloadResult struct {
		Index int
		Data  []byte
		Ext   string
	}

	results := make([]downloadResult, len(imageUrls))
	g, gCtx := errgroup.WithContext(ctx)

	// Limit concurrency to avoid getting banned
	g.SetLimit(5)

	for i, url := range imageUrls {
		i, url := i, url // capture loop variables
		g.Go(func() error {
			req, err := http.NewRequestWithContext(gCtx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}

			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
			req.Header.Set("Referer", chapter.URL)

			resp, err := a.HTTPClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to download image %d: %w", i, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status %d for image %d", resp.StatusCode, i)
			}

			if contentLength := resp.ContentLength; contentLength > maxImageBytes {
				return fmt.Errorf("image %d exceeds max size (%d bytes)", i, maxImageBytes)
			}

			data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
			if err != nil {
				return fmt.Errorf("failed to read body for image %d: %w", i, err)
			}
			if len(data) > maxImageBytes {
				return fmt.Errorf("image %d exceeds max size (%d bytes)", i, maxImageBytes)
			}

			ext := ".jpg"
			if ctype := resp.Header.Get("Content-Type"); ctype == "image/png" {
				ext = ".png"
			} else if ctype == "image/webp" {
				ext = ".webp"
			}

			results[i] = downloadResult{
				Index: i,
				Data:  data,
				Ext:   ext,
			}
			return nil
		})
	}

	if err = g.Wait(); err != nil {
		return fmt.Errorf("failed to download chapter images: %w", err)
	}

	var zipMu sync.Mutex
	for _, res := range results {
		filename := fmt.Sprintf("%03d%s", res.Index+1, res.Ext)

		zipMu.Lock()
		writer, wErr := zipWriter.Create(filename)
		if wErr == nil {
			_, wErr = writer.Write(res.Data)
		}
		zipMu.Unlock()

		if wErr != nil {
			err = fmt.Errorf("failed to write to zip: %w", wErr)
			return err
		}
	}

	if err = zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize zip archive: %w", err)
	}

	if err = os.Rename(tmpPath, cbzPath); err != nil {
		return fmt.Errorf("failed to save final cbz archive: %w", err)
	}

	return nil
}

// sanitizePath removes invalid characters for filenames
func sanitizePath(name string) string {
	invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	sanitized := name
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}
	return strings.TrimSpace(sanitized)
}
