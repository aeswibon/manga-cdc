package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: fetch-title <mangaplus-title-id>")
		os.Exit(2)
	}
	titleID := os.Args[1]

	ctx := context.Background()
	mp := adapter.NewMangaPlusAdapter()
	series, err := mp.FetchSeries(ctx, titleID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FetchSeries: %v\n", err)
		os.Exit(1)
	}
	chapters, err := mp.FetchChapters(ctx, titleID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FetchChapters: %v\n", err)
		os.Exit(1)
	}
	sort.Slice(chapters, func(i, j int) bool { return chapters[i].Number < chapters[j].Number })

	out := map[string]any{
		"series":        series,
		"chapter_count": len(chapters),
		"chapters":      chapters,
	}
	if len(chapters) > 0 {
		out["latest"] = chapters[len(chapters)-1]
		out["seed_chapters"] = chapters[:len(chapters)-1]
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}
