const MANGADEX_HOST = 'uploads.mangadex.org';
const MANGAPLUS_HOSTS = ['jumpg-assets.tokyo-cdn.com', 'img.mangaplus.shueisha.co.jp'];

export const MAX_COVER_BYTES = 4 * 1024 * 1024;

export function isAllowedCoverUrl(raw: string): boolean {
  let parsed: URL;
  try {
    parsed = new URL(raw.trim());
  } catch {
    return false;
  }
  if (parsed.protocol !== 'https:' || !parsed.hostname) {
    return false;
  }
  if (parsed.username || parsed.password) {
    return false;
  }
  return true;
}

function isMangaDexThumbnailPath(pathname: string): boolean {
  return /\.(256|512)\.jpg$/i.test(pathname);
}

function isImagePath(pathname: string): boolean {
  return /\.(png|jpe?g|webp|gif)$/i.test(pathname);
}

/** Prefer MangaDex CDN thumbnails so proxied covers stay under Vercel response limits. */
export function coverFetchCandidates(raw: string): string[] {
  const target = new URL(raw.trim());
  const host = target.hostname.toLowerCase();

  if (host === MANGADEX_HOST) {
    if (isMangaDexThumbnailPath(target.pathname)) {
      return [target.toString()];
    }
    if (isImagePath(target.pathname)) {
      const base = target.toString();
      return [`${base}.512.jpg`, `${base}.256.jpg`, base];
    }
  }

  return [target.toString()];
}

export function coverFetchReferer(raw: string): string {
  const host = new URL(raw.trim()).hostname.toLowerCase();
  if (MANGAPLUS_HOSTS.some((allowed) => host === allowed || host.endsWith(`.${allowed}`))) {
    return 'https://mangaplus.shueisha.co.jp/';
  }
  if (host === MANGADEX_HOST) {
    return 'https://mangadex.org/';
  }
  return 'https://mangadex.org/';
}

export function coverFetchHeaders(raw: string): Record<string, string> {
  return {
    Accept: 'image/*',
    Referer: coverFetchReferer(raw),
    'User-Agent': 'manga-cdc-dashboard/1.0',
  };
}
