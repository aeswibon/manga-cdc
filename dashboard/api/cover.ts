export const config = { runtime: 'edge' };

import {
  MAX_COVER_BYTES,
  MAX_COVER_REDIRECTS,
  coverFetchCandidates,
  coverFetchHeaders,
  isAllowedCoverUrl,
  validateRedirectLocation,
} from '../src/cover-proxy.js';

async function fetchCover(url: string): Promise<{ body: ArrayBuffer; contentType: string } | null> {
  let current = new URL(url);

  for (let hop = 0; hop <= MAX_COVER_REDIRECTS; hop++) {
    const upstream = await fetch(current.toString(), {
      headers: coverFetchHeaders(current.toString()),
      redirect: 'manual',
    });

    if (upstream.status >= 300 && upstream.status < 400) {
      const location = upstream.headers.get('location');
      if (!location) {
        return null;
      }
      const next = validateRedirectLocation(current, location);
      if (!next) {
        return null;
      }
      current = next;
      continue;
    }

    if (!upstream.ok) {
      return null;
    }

    const contentType = upstream.headers.get('content-type') ?? 'application/octet-stream';
    if (!contentType.startsWith('image/')) {
      return null;
    }

    const lengthHeader = upstream.headers.get('content-length');
    if (lengthHeader && Number(lengthHeader) > MAX_COVER_BYTES) {
      return null;
    }

    const arrayBuffer = await upstream.arrayBuffer();
    if (arrayBuffer.byteLength > MAX_COVER_BYTES) {
      return null;
    }

    return { body: arrayBuffer, contentType };
  }

  return null;
}

export default async function handler(req: Request) {
  if ((req.method ?? 'GET').toUpperCase() !== 'GET') {
    return Response.json({ error: 'Method not allowed' }, { status: 405 });
  }

  const url = new URL(req.url);
  const value = url.searchParams.get('url');

  if (!value || !isAllowedCoverUrl(value)) {
    return Response.json({ error: 'Invalid cover URL' }, { status: 400 });
  }

  try {
    for (const candidate of coverFetchCandidates(value)) {
      const cover = await fetchCover(candidate);
      if (!cover) {
        continue;
      }

      return new Response(cover.body, {
        status: 200,
        headers: {
          'Content-Type': cover.contentType,
          'Cache-Control': 'public, max-age=86400, s-maxage=604800, stale-while-revalidate=2592000',
        },
      });
    }

    return Response.json({ error: 'Cover fetch failed' }, { status: 502 });
  } catch {
    return Response.json({ error: 'Cover fetch failed' }, { status: 502 });
  }
}
