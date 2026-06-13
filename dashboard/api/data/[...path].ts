import type { VercelRequest, VercelResponse } from '@vercel/node';
import { buildTargetPath, proxyNotifierGet, proxySegments } from './_proxy.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  const targetPath = buildTargetPath(proxySegments(req));
  await proxyNotifierGet(req, res, targetPath);
}
