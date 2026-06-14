export const config = { runtime: 'edge' };
import { buildTargetPath, proxyNotifierGet, proxySegments } from './_proxy.js';

export default async function handler(req: Request) {
  const targetPath = buildTargetPath(proxySegments(req));
  return await proxyNotifierGet(req, targetPath);
}
