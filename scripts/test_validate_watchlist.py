#!/usr/bin/env python3
"""Unit tests for scripts/validate-watchlist.py."""

from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path

_MODULE_PATH = Path(__file__).resolve().parent / "validate-watchlist.py"
_SPEC = importlib.util.spec_from_file_location("validate_watchlist", _MODULE_PATH)
assert _SPEC and _SPEC.loader
validate_watchlist = importlib.util.module_from_spec(_SPEC)
_SPEC.loader.exec_module(validate_watchlist)

lint_watchlist = validate_watchlist.lint_watchlist
lint_watchlist_file = validate_watchlist.lint_watchlist_file


class ValidateWatchlistTests(unittest.TestCase):
    def lint_yaml(self, content: str) -> list[str]:
        with tempfile.NamedTemporaryFile("w", suffix=".yaml", delete=False) as handle:
            handle.write(content)
            path = Path(handle.name)
        self.addCleanup(path.unlink, missing_ok=True)
        ok, errors = lint_watchlist_file(path)
        self.assertFalse(ok)
        return errors

    def test_valid_mangadex_entry_passes(self) -> None:
        entries = [
            {
                "source": "mangadex",
                "source_id": "a1c7c817-4e59-43b7-9365-09675a149a6f",
                "title": "One Piece",
                "source_url": (
                    "https://mangadex.org/title/"
                    "a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece"
                ),
            }
        ]
        self.assertEqual(lint_watchlist(entries), [])

    def test_rejects_wrong_source_url_host(self) -> None:
        errors = self.lint_yaml(
            """- source: mangadex
  source_id: a1c7c817-4e59-43b7-9365-09675a149a6f
  title: One Piece
  source_url: https://example.com/title/a1c7c817-4e59-43b7-9365-09675a149a6f
"""
        )
        self.assertTrue(any("source_url host" in error for error in errors))

    def test_rejects_mangadex_source_id_not_in_url(self) -> None:
        errors = self.lint_yaml(
            """- source: mangadex
  source_id: a1c7c817-4e59-43b7-9365-09675a149a6f
  title: One Piece
  source_url: https://mangadex.org/title/00000000-0000-0000-0000-000000000000/one-piece
"""
        )
        self.assertTrue(any("source_url should include source_id" in error for error in errors))

    def test_rejects_invalid_mangaplus_url(self) -> None:
        errors = self.lint_yaml(
            """- source: mangaplus
  source_id: "100127"
  title: SAKAMOTO DAYS
  source_url: https://mangaplus.shueisha.co.jp/viewer/100127
"""
        )
        self.assertTrue(any("/titles/100127" in error for error in errors))

    def test_rejects_fallback_same_source_as_primary(self) -> None:
        errors = self.lint_yaml(
            """- source: mangadex
  source_id: a1c7c817-4e59-43b7-9365-09675a149a6f
  title: One Piece
  source_url: https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece
  fallback_sources:
    - source: mangafire
      source_id: one-piece
      source_url: https://mangafire.to/manga/one-piece
    - source: mangadex
      source_id: b2d8c918-5f6a-54c8-b47e-3c5853f58555
"""
        )
        self.assertTrue(any("must differ from primary source" in error for error in errors))

    def test_rejects_duplicate_fallback_pair(self) -> None:
        errors = self.lint_yaml(
            """- source: mangadex
  source_id: a1c7c817-4e59-43b7-9365-09675a149a6f
  title: One Piece
  source_url: https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece
  fallback_sources:
    - source: mangafire
      source_id: one-piece
    - source: mangafire
      source_id: one-piece
"""
        )
        self.assertTrue(any("duplicate fallback source+source_id" in error for error in errors))

    def test_rejects_invalid_status(self) -> None:
        errors = self.lint_yaml(
            """- source: mangadex
  source_id: a1c7c817-4e59-43b7-9365-09675a149a6f
  title: One Piece
  source_url: https://mangadex.org/title/a1c7c817-4e59-43b7-9365-09675a149a6f/one-piece
  status: RUNNING
"""
        )
        self.assertTrue(any("invalid status" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
