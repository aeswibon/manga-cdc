# Contributing to the Community Watchlist

The manga-cdc dashboard is **read-only** for public users. Series are tracked from a community-curated list in [`data/watchlist.yaml`](data/watchlist.yaml). To add or update tracked manga, open a pull request that edits that file.

## Adding a series

1. Fork the repository and create a branch.
2. Add an entry to `data/watchlist.yaml` (or edit an existing one).
3. Run the validator locally before pushing:

   ```bash
   pip install pyyaml
   python scripts/validate-watchlist.py
   ```

4. Open a pull request. CI runs the same validation on every change to the watchlist.

## Entry format

Each entry is a YAML object with these **required** fields:

| Field | Description |
|-------|-------------|
| `source` | Scraper adapter name (see valid sources below) |
| `source_id` | Unique ID for that source (e.g. MangaDex UUID) |
| `title` | Human-readable series title |
| `source_url` | Full HTTP(S) URL to the series page on the source site |

Example:

```yaml
- source: mangadex
  source_id: a1c3b275-c93f-4279-a17d-2b4742e47444
  title: One Piece
  source_url: https://mangadex.org/title/a1c3b275-c93f-4279-a17d-2b4742e47444/one-piece
```

## Valid sources

| `source` value | Site |
|----------------|------|
| `mangadex` | [MangaDex](https://mangadex.org) |
| `mangaplus` | [Manga Plus](https://mangaplus.shueisha.co.jp) |
| `mangafire` | MangaFire |
| `asurascans` | Asura Scans |
| `mangapill` | MangaPill |
| `mangatown` | MangaTown |

## Validation rules

The CI script (`scripts/validate-watchlist.py`) checks that:

- The file parses as YAML and contains a non-empty list of entries
- Every entry has all required fields with non-empty string values
- `source` is one of the valid adapter names above
- `source_url` is a valid HTTP or HTTPS URL
- No duplicate `source` + `source_id` pairs exist
- No duplicate `title` values exist (same title must use one canonical `source_id`)

## Removing a series

Delete the corresponding entry from `data/watchlist.yaml` and open a PR with a brief reason. On the next watchlist sync, the scraper removes that series from the database (chapters and notification logs cascade automatically).
