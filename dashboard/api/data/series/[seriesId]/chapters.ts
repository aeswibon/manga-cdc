export const config = { runtime: 'edge' };
import { proxyNotifierGet } from '../../_proxy.js';

function seriesIdFromRequest(req: Request): string | undefined {
  const url = new URL(req.url);
  const parts = url.pathname.split('/');
  const idx = parts.indexOf('series');
  if (idx >= 0 && idx + 1 < parts.length) {
    return parts[idx + 1];
  }
  return undefined;
}

export default async function handler(req: Request) {
  const seriesId = seriesIdFromRequest(req);
  if (!seriesId) {
    return Response.json({ error: 'seriesId required' }, { status: 400 });
  }

  return await proxyNotifierGet(req, `/api/series/${encodeURIComponent(seriesId)}/chapters`);
}
