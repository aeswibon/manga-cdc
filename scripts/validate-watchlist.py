#!/usr/bin/env python3
"""Validate and lint data/watchlist.yaml structure and content."""

from __future__ import annotations

import argparse
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

VALID_STATUSES = frozenset({
    "ONGOING",
    "COMPLETED",
    "HIATUS",
    "CANCELLED",
})

SOURCE_URL_HOSTS: dict[str, tuple[str, ...]] = {
    "mangadex": ("mangadex.org",),
    "mangaplus": ("mangaplus.shueisha.co.jp",),
    "mangafire": ("mangafire.to", "www.mangafire.to"),
    "asurascans": ("asurascans.com", "www.asurascans.com"),
    "mangapill": ("mangapill.com", "www.mangapill.com"),
    "mangatown": ("mangatown.com", "www.mangatown.com"),
}

REQUIRED_FIELDS = ("source", "source_id", "title", "source_url")
HTTP_URL_PATTERN = re.compile(r"^https?://", re.IGNORECASE)
MANGADEX_UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
    re.IGNORECASE,
)


def is_http_url(value: str) -> bool:
    if not HTTP_URL_PATTERN.match(value):
        return False
    parsed = urlparse(value)
    return bool(parsed.netloc)


def url_hostname(value: str) -> str:
    return urlparse(value.strip()).hostname.lower() if value.strip() else ""


def lint_source_url_host(source: str, source_url: str, prefix: str) -> list[str]:
    errors: list[str] = []
    allowed = SOURCE_URL_HOSTS.get(source)
    if allowed is None:
        return errors

    host = url_hostname(source_url)
    if host not in allowed:
        errors.append(
            f"{prefix}: source_url host '{host or '(missing)'}' does not match source "
            f"'{source}' (expected: {', '.join(allowed)})"
        )
    return errors


def lint_source_id_in_url(source: str, source_id: str, source_url: str, prefix: str) -> list[str]:
    errors: list[str] = []
    normalized_id = source_id.strip()
    normalized_url = source_url.strip().lower()

    if source == "mangadex":
        if not MANGADEX_UUID_PATTERN.fullmatch(normalized_id):
            errors.append(f"{prefix}: mangadex source_id must be a UUID")
        elif normalized_id.lower() not in normalized_url:
            errors.append(
                f"{prefix}: mangadex source_url should include source_id '{normalized_id}'"
            )
    elif source == "mangaplus":
        if not normalized_id.isdigit():
            errors.append(f"{prefix}: mangaplus source_id must be numeric")
        elif f"/titles/{normalized_id}" not in normalized_url:
            errors.append(
                f"{prefix}: mangaplus source_url should include '/titles/{normalized_id}'"
            )

    return errors


def validate_notifications(notifications: object, prefix: str) -> list[str]:
    errors: list[str] = []
    if not isinstance(notifications, dict):
        return [f"{prefix}: 'notifications' must be a mapping"]

    notify_every = notifications.get("notify_every")
    if notify_every is not None and not isinstance(notify_every, int):
        errors.append(f"{prefix}: 'notifications.notify_every' must be an integer")
    if isinstance(notify_every, int) and notify_every < 0:
        errors.append(f"{prefix}: 'notifications.notify_every' must be >= 0")

    for field in ("preferred_groups", "blocked_groups"):
        groups = notifications.get(field)
        if groups is None:
            continue
        if not isinstance(groups, list):
            errors.append(f"{prefix}: 'notifications.{field}' must be a list of strings")
            continue
        for index, group in enumerate(groups):
            if not isinstance(group, str) or not group.strip():
                errors.append(
                    f"{prefix}: 'notifications.{field}[{index}]' must be a non-empty string"
                )

    block_early_week = notifications.get("block_early_week")
    if block_early_week is not None and not isinstance(block_early_week, bool):
        errors.append(f"{prefix}: 'notifications.block_early_week' must be a boolean")

    return errors


def validate_fallback_sources(
    fallback_sources: object,
    prefix: str,
    primary_source: str,
    primary_source_id: str,
) -> list[str]:
    errors: list[str] = []
    if not isinstance(fallback_sources, list):
        return [f"{prefix}: 'fallback_sources' must be a list"]

    seen_fallback_keys: set[tuple[str, str]] = set()
    primary_key = (primary_source.strip(), primary_source_id.strip())

    for index, item in enumerate(fallback_sources):
        item_prefix = f"{prefix}: fallback_sources[{index}]"
        if not isinstance(item, dict):
            errors.append(f"{item_prefix}: must be a mapping")
            continue

        source = item.get("source")
        source_id = item.get("source_id")
        if not isinstance(source, str) or not source.strip():
            errors.append(f"{item_prefix}: 'source' must be a non-empty string")
            continue
        source = source.strip()

        if source not in VALID_SOURCES:
            errors.append(
                f"{item_prefix}: invalid source '{source}' "
                f"(allowed: {', '.join(sorted(VALID_SOURCES))})"
            )

        if not isinstance(source_id, str) or not source_id.strip():
            errors.append(f"{item_prefix}: 'source_id' must be a non-empty string")
            continue
        source_id = source_id.strip()

        if source == primary_source.strip():
            errors.append(
                f"{item_prefix}: fallback source '{source}' must differ from primary source"
            )

        fallback_key = (source, source_id)
        if fallback_key == primary_key:
            errors.append(
                f"{item_prefix}: fallback source+source_id duplicates the primary entry"
            )
        if fallback_key in seen_fallback_keys:
            errors.append(
                f"{item_prefix}: duplicate fallback source+source_id ({source}, {source_id})"
            )
        seen_fallback_keys.add(fallback_key)

        source_url = item.get("source_url")
        if source_url is not None and isinstance(source_url, str) and source_url.strip():
            if not is_http_url(source_url.strip()):
                errors.append(f"{item_prefix}: 'source_url' must be an HTTP or HTTPS URL")
            else:
                errors.extend(lint_source_url_host(source, source_url, item_prefix))
                errors.extend(
                    lint_source_id_in_url(source, source_id, source_url, item_prefix)
                )

    return errors


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
    source_id = entry.get("source_id")
    source_url = entry.get("source_url")

    if isinstance(source, str):
        source = source.strip()
        if source not in VALID_SOURCES:
            errors.append(
                f"{prefix}: invalid source '{source}' "
                f"(allowed: {', '.join(sorted(VALID_SOURCES))})"
            )

    if isinstance(source_url, str) and source_url.strip():
        if not is_http_url(source_url.strip()):
            errors.append(f"{prefix}: 'source_url' must be an HTTP or HTTPS URL")
        elif isinstance(source, str) and source in VALID_SOURCES:
            errors.extend(lint_source_url_host(source, source_url, prefix))

    status = entry.get("status")
    if status is not None:
        if not isinstance(status, str) or not status.strip():
            errors.append(f"{prefix}: 'status' must be a non-empty string when set")
        elif status.strip().upper() not in VALID_STATUSES:
            errors.append(
                f"{prefix}: invalid status '{status}' "
                f"(allowed: {', '.join(sorted(VALID_STATUSES))})"
            )

    if (
        isinstance(source, str)
        and source in VALID_SOURCES
        and isinstance(source_id, str)
        and isinstance(source_url, str)
        and source_id.strip()
        and source_url.strip()
        and is_http_url(source_url.strip())
    ):
        errors.extend(lint_source_id_in_url(source, source_id, source_url, prefix))

    notifications = entry.get("notifications")
    if notifications is not None:
        errors.extend(validate_notifications(notifications, prefix))

    fallback_sources = entry.get("fallback_sources")
    if fallback_sources is not None:
        if isinstance(source, str) and isinstance(source_id, str):
            errors.extend(
                validate_fallback_sources(
                    fallback_sources,
                    prefix,
                    source,
                    source_id,
                )
            )
        else:
            errors.extend(validate_fallback_sources(fallback_sources, prefix, "", ""))

    return errors


def lint_watchlist(entries: list[object]) -> list[str]:
    errors: list[str] = []
    seen_keys: dict[tuple[str, str], int] = {}
    seen_titles: dict[str, list[int]] = {}

    for index, entry in enumerate(entries):
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
        sample_title = entries[indices[0]].get("title", "")
        errors.append(
            f"duplicate title '{sample_title}' at entries {labels} — "
            "keep one canonical source_id per title (different links create separate trackers)"
        )

    return errors


def load_watchlist(path: Path) -> list[object] | None:
    if not path.is_file():
        print(f"error: watchlist file not found: {path}", file=sys.stderr)
        return None

    try:
        with path.open(encoding="utf-8") as handle:
            data = yaml.safe_load(handle)
    except yaml.YAMLError as exc:
        print(f"error: failed to parse YAML: {exc}", file=sys.stderr)
        return None

    if data is None:
        print("error: watchlist is empty", file=sys.stderr)
        return None

    if not isinstance(data, list):
        print("error: watchlist root must be a list of entries", file=sys.stderr)
        return None

    if len(data) == 0:
        print("error: watchlist must contain at least one entry", file=sys.stderr)
        return None

    return data


def lint_watchlist_file(path: Path) -> tuple[bool, list[str]]:
    entries = load_watchlist(path)
    if entries is None:
        return False, ["failed to load watchlist"]

    errors = lint_watchlist(entries)
    return len(errors) == 0, errors


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate and lint watchlist.yaml")
    parser.add_argument(
        "--path",
        type=Path,
        default=None,
        help="Path to watchlist YAML (default: data/watchlist.yaml)",
    )
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parent.parent
    watchlist_path = args.path or (repo_root / "data" / "watchlist.yaml")

    ok, errors = lint_watchlist_file(watchlist_path)
    if not ok:
        if errors == ["failed to load watchlist"]:
            return 1
        print("watchlist validation failed:", file=sys.stderr)
        for error in errors:
            print(f"  - {error}", file=sys.stderr)
        return 1

    entries = load_watchlist(watchlist_path)
    count = len(entries) if entries is not None else 0
    print(f"watchlist OK ({count} entries)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
