#!/usr/bin/env python3
"""Validate data/watchlist.yaml structure and content."""

from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import urlparse

try:
    import yaml
except ImportError:
    print("error: PyYAML is required (pip install pyyaml)", file=sys.stderr)
    sys.exit(1)

VALID_SOURCES = frozenset({
    "mangadex",
    "mangaplus",
    "mangafire",
    "asurascans",
    "mangapill",
    "mangatown",
})

REQUIRED_FIELDS = ("source", "source_id", "title", "source_url")
HTTP_URL_PATTERN = re.compile(r"^https?://", re.IGNORECASE)


def is_http_url(value: str) -> bool:
    if not HTTP_URL_PATTERN.match(value):
        return False
    parsed = urlparse(value)
    return bool(parsed.netloc)


def validate_entry(entry: object, index: int) -> list[str]:
    errors: list[str] = []
    prefix = f"entry #{index + 1}"

    if not isinstance(entry, dict):
        return [f"{prefix}: must be a mapping"]

    for field in REQUIRED_FIELDS:
        if field not in entry:
            errors.append(f"{prefix}: missing required field '{field}'")
            continue
        value = entry[field]
        if not isinstance(value, str) or not value.strip():
            errors.append(f"{prefix}: '{field}' must be a non-empty string")

    source = entry.get("source")
    if isinstance(source, str):
        if source not in VALID_SOURCES:
            errors.append(
                f"{prefix}: invalid source '{source}' "
                f"(allowed: {', '.join(sorted(VALID_SOURCES))})"
            )

    source_url = entry.get("source_url")
    if isinstance(source_url, str) and source_url.strip():
        if not is_http_url(source_url.strip()):
            errors.append(f"{prefix}: 'source_url' must be an HTTP or HTTPS URL")

    return errors


def main() -> int:
    repo_root = Path(__file__).resolve().parent.parent
    watchlist_path = repo_root / "data" / "watchlist.yaml"

    if not watchlist_path.is_file():
        print(f"error: watchlist file not found: {watchlist_path}", file=sys.stderr)
        return 1

    try:
        with watchlist_path.open(encoding="utf-8") as handle:
            data = yaml.safe_load(handle)
    except yaml.YAMLError as exc:
        print(f"error: failed to parse YAML: {exc}", file=sys.stderr)
        return 1

    if data is None:
        print("error: watchlist is empty", file=sys.stderr)
        return 1

    if not isinstance(data, list):
        print("error: watchlist root must be a list of entries", file=sys.stderr)
        return 1

    if len(data) == 0:
        print("error: watchlist must contain at least one entry", file=sys.stderr)
        return 1

    errors: list[str] = []
    seen_keys: dict[tuple[str, str], int] = {}
    seen_titles: dict[str, list[int]] = {}

    for index, entry in enumerate(data):
        errors.extend(validate_entry(entry, index))

        if isinstance(entry, dict):
            source = entry.get("source")
            source_id = entry.get("source_id")
            title = entry.get("title")
            if (
                isinstance(source, str)
                and isinstance(source_id, str)
                and source.strip()
                and source_id.strip()
            ):
                key = (source.strip(), source_id.strip())
                if key in seen_keys:
                    errors.append(
                        f"entry #{index + 1}: duplicate source+source_id pair "
                        f"({source}, {source_id}) — first seen at entry #{seen_keys[key] + 1}"
                    )
                else:
                    seen_keys[key] = index

            if isinstance(title, str) and title.strip():
                normalized_title = title.strip().casefold()
                seen_titles.setdefault(normalized_title, []).append(index)

    for indices in seen_titles.values():
        if len(indices) < 2:
            continue
        labels = ", ".join(f"#{i + 1}" for i in indices)
        sample_title = data[indices[0]].get("title", "")
        errors.append(
            f"duplicate title '{sample_title}' at entries {labels} — "
            "keep one canonical source_id per title (different links create separate trackers)"
        )

    if errors:
        print("watchlist validation failed:", file=sys.stderr)
        for error in errors:
            print(f"  - {error}", file=sys.stderr)
        return 1

    print(f"watchlist OK ({len(data)} entries)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
