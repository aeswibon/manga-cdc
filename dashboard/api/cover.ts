import type { VercelRequest, VercelResponse } from '@vercel/node';
import dns from 'node:dns/promises';
import net from 'node:net';
import {
  MAX_COVER_BYTES,
  coverFetchCandidates,
  coverFetchHeaders,
  isAllowedCoverUrl,
} from '../src/cover-proxy.js';

function isPrivateIp(ip: string): boolean {
  if (net.isIPv4(ip)) {
    const parts = ip.split('.').map(Number);
    if (parts[0] === 10) return true;
    if (parts[0] === 127) return true;
    if (parts[0] === 169 && parts[1] === 254) return true;
    if (parts[0] === 192 && parts[1] === 168) return true;
    if (parts[0] === 172 && parts[1] >= 16 && parts[1] <= 31) return true;
    return false;
  }
  if (ip === '::1' || ip.startsWith('fe80:') || ip.startsWith('fc') || ip.startsWith('fd')) {
    return true;
  }
  return false;
}

async function assertPublicHost(hostname: string): Promise<void> {
  const host = hostname.toLowerCase();
  if (host === 'localhost' || host.endsWith('.local')) {
    throw new Error('blocked host');
  }
  const records = await dns.lookup(host, { all: true, verbatim: true });
  for (const record of records) {
    if (isPrivateIp(record.address)) {
      throw new Error('blocked host');
    }
  }
}

async function fetchCover(url: string): Promise<{ body: Buffer; contentType: string } | null> {
  const upstream = await fetch(url, {
    headers: coverFetchHeaders(url),
    redirect: 'follow',
  });

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

  const body = Buffer.from(await upstream.arrayBuffer());
  if (body.byteLength > MAX_COVER_BYTES) {
    return null;
  }

  return { body, contentType };
}

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if ((req.method ?? 'GET').toUpperCase() !== 'GET') {
    res.status(405).json({ error: 'Method not allowed' });
    return;
  }

  const raw = req.query.url;
  const value = Array.isArray(raw) ? raw[0] : raw;
  if (!value || !isAllowedCoverUrl(value)) {
    res.status(400).json({ error: 'Invalid cover URL' });
    return;
  }

  const target = new URL(value.trim());

  try {
    await assertPublicHost(target.hostname);
  } catch {
    res.status(400).json({ error: 'Cover URL host is not allowed' });
    return;
  }

  try {
    for (const candidate of coverFetchCandidates(value)) {
      const cover = await fetchCover(candidate);
      if (!cover) {
        continue;
      }

      res.setHeader('Content-Type', cover.contentType);
      res.setHeader('Cache-Control', 'public, max-age=86400, stale-while-revalidate=604800');
      res.status(200).send(cover.body);
      return;
    }

    res.status(502).json({ error: 'Cover fetch failed' });
  } catch {
    res.status(502).json({ error: 'Cover fetch failed' });
  }
}
