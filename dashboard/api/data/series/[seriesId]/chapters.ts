import type { VercelRequest, VercelResponse } from '@vercel/node';
import { proxyNotifierGet } from '../../_proxy.js';

function seriesIdFromRequest(req: VercelRequest): string | undefined {
  const raw = req.query.seriesId;
  const value = Array.isArray(raw) ? raw[0] : raw;
  return typeof value === 'string' && value.length > 0 ? value : undefined;
}

export default async function handler(req: VercelRequest, res: VercelResponse) {
  const seriesId = seriesIdFromRequest(req);
  if (!seriesId) {
    res.status(400).json({ error: 'seriesId required' });
    return;
  }

  await proxyNotifierGet(req, res, `/api/series/${encodeURIComponent(seriesId)}/chapters`);
}
