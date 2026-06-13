import { describe, expect, test } from 'bun:test';
import { coverFetchCandidates, coverFetchReferer, isAllowedCoverUrl } from './cover-proxy';

describe('cover proxy helpers', () => {
  test('isAllowedCoverUrl accepts https image URLs', () => {
    expect(isAllowedCoverUrl('https://uploads.mangadex.org/covers/a/b.png')).toBe(true);
    expect(isAllowedCoverUrl('http://uploads.mangadex.org/covers/a/b.png')).toBe(false);
    expect(isAllowedCoverUrl('not-a-url')).toBe(false);
  });

  test('coverFetchCandidates prefer MangaDex thumbnails before original', () => {
    const original =
      'https://uploads.mangadex.org/covers/a1c7c817-4e59-43b7-9365-09675a149a6f/2f4aca53-64c7-46ac-ae85-3bc9b3169890.png';
    expect(coverFetchCandidates(original)).toEqual([
      `${original}.512.jpg`,
      `${original}.256.jpg`,
      original,
    ]);
  });

  test('coverFetchCandidates leave existing thumbnails unchanged', () => {
    const thumb =
      'https://uploads.mangadex.org/covers/a1c7c817-4e59-43b7-9365-09675a149a6f/2f4aca53-64c7-46ac-ae85-3bc9b3169890.png.512.jpg';
    expect(coverFetchCandidates(thumb)).toEqual([thumb]);
  });

  test('coverFetchReferer uses source-specific referers', () => {
    expect(coverFetchReferer('https://uploads.mangadex.org/covers/a/b.png')).toBe('https://mangadex.org/');
    expect(
      coverFetchReferer(
        'https://jumpg-assets.tokyo-cdn.com/secure/title/100127/title_thumbnail_main/313624.webp?hash=abc',
      ),
    ).toBe('https://mangaplus.shueisha.co.jp/');
  });
});
