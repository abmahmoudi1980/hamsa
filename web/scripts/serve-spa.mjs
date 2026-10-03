/**
 * Minimal static server for the built SPA, with the fallback behaviour the
 * production host must provide.
 *
 * Why this exists: `adapter({ fallback: 'index.html' })` writes the SPA shell to
 * `build/index.html`, but a plain static file server answers 404 for any path it
 * does not recognise. Without a rewrite, a deep link like `/m/buildings` — which
 * the manager console produces on every navigation, and which users will
 * bookmark or paste — would break on hard refresh.
 *
 * Production handles this in nginx (`try_files $uri $uri/ /index.html`, see
 * deploy/nginx-hamsa.conf); this script mirrors it for `npm run preview` and for
 * the Playwright E2E run, so what CI exercises is what production does.
 *
 * Deliberately dependency-free and read-only. Usage:
 *   node scripts/serve-spa.mjs [--port 4173] [--dir build]
 */
import { createReadStream } from 'node:fs';
import { stat } from 'node:fs/promises';
import { createServer, request as proxyRequest } from 'node:http';
import { extname, join, normalize, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = fileURLToPath(new URL('.', import.meta.url));

function arg(name, fallback) {
	const index = process.argv.indexOf(`--${name}`);
	return index !== -1 && process.argv[index + 1] ? process.argv[index + 1] : fallback;
}

const port = Number(arg('port', process.env.PORT ?? 4173));
const root = resolve(here, '..', arg('dir', 'build'));
const apiOrigin = new URL(process.env.HAMSA_API_ORIGIN ?? 'http://localhost:8080');

const MIME = {
	'.html': 'text/html; charset=utf-8',
	'.js': 'text/javascript; charset=utf-8',
	'.mjs': 'text/javascript; charset=utf-8',
	'.css': 'text/css; charset=utf-8',
	'.json': 'application/json; charset=utf-8',
	'.svg': 'image/svg+xml',
	'.png': 'image/png',
	'.jpg': 'image/jpeg',
	'.jpeg': 'image/jpeg',
	'.webp': 'image/webp',
	'.ico': 'image/x-icon',
	'.woff2': 'font/woff2',
	'.woff': 'font/woff',
	'.ttf': 'font/ttf',
	'.pdf': 'application/pdf',
	'.txt': 'text/plain; charset=utf-8',
	'.map': 'application/json; charset=utf-8'
};

/**
 * Resolves a URL path to a file inside root, or null when it escapes.
 * Traversal guard: a normalised path that no longer starts with root is refused.
 */
function resolveInRoot(pathname) {
	let decoded;
	try {
		decoded = decodeURIComponent(pathname);
	} catch {
		return null;
	}
	const candidate = normalize(join(root, decoded));
	if (candidate !== root && !candidate.startsWith(root + sep)) return null;
	return candidate;
}

async function statFile(path) {
	try {
		const info = await stat(path);
		// isFile(), not isDirectory(): serving a directory would stream a
		// directory listing and then fail with EISDIR.
		return info.isFile() ? info : null;
	} catch {
		return null;
	}
}

const server = createServer(async (req, res) => {
	if (!req.url) {
		res.writeHead(400).end('Bad Request');
		return;
	}

	const { pathname } = new URL(req.url, 'http://localhost');
	if (pathname.startsWith('/api/') || pathname.startsWith('/files/')) {
		const upstream = proxyRequest(new URL(req.url, apiOrigin), {
			method: req.method,
			headers: { ...req.headers, host: apiOrigin.host }
		});
		upstream.on('response', (upstreamResponse) => {
			res.writeHead(upstreamResponse.statusCode ?? 502, upstreamResponse.headers);
			upstreamResponse.pipe(res);
		});
		upstream.on('error', () => {
			if (!res.headersSent) {
				res
					.writeHead(502, { 'Content-Type': MIME['.json'] })
					.end(
						JSON.stringify({ error: { code: 'INTERNAL', message: 'ارتباط با سرور برقرار نشد.' } })
					);
			} else {
				res.destroy();
			}
		});
		req.pipe(upstream);
		return;
	}

	const target = resolveInRoot(pathname);
	if (target === null) {
		res.writeHead(403).end('Forbidden');
		return;
	}

	// An extension means a real asset request: a miss is a genuine 404, not a
	// route. Only extensionless paths fall through to the SPA shell, so a missing
	// hashed asset still reports honestly.
	let file = await statFile(target);
	let filePath = file ? target : null;
	if (!filePath && !extname(pathname)) {
		const shell = join(root, 'index.html');
		file = await statFile(shell);
		filePath = file ? shell : null;
	}

	if (!filePath || !file) {
		res.writeHead(404, { 'Content-Type': MIME['.txt'] }).end('Not Found');
		return;
	}

	const ext = extname(filePath).toLowerCase();
	// SvelteKit names chunks `<hash>.js` and assets `assets/<n>.<hash>.css`, so the
	// content hash is the last dot-segment of the stem. `fonts/` is deliberately
	// excluded: Vazirmatn files are named by weight, not content, so caching them
	// immutably would serve a stale subset for a year after a re-subset.
	// Vite guarantees every file under `_app/immutable/` is content-hashed and safe
	// to cache forever; everything else (index.html, robots.txt, version.json, and
	// the weight-named fonts) is not. Testing the directory is exact, whereas
	// guessing from the filename is not: Vite hashes are base64-ish and a chunk can
	// come out as `DSKyKlGf` with no digit at all.
	const isContentHashed = filePath.includes(`${sep}_app${sep}immutable${sep}`);

	const headers = {
		'Content-Type': MIME[ext] ?? 'application/octet-stream',
		'Content-Length': file.size
	};
	// Content-hashed assets are immutable, so they can be cached hard; the HTML
	// shell must not be, or a deploy would never reach the browser.
	headers['Cache-Control'] = isContentHashed
		? 'public, max-age=31536000, immutable'
		: ext === '.html'
			? 'no-cache'
			: 'public, max-age=3600';

	res.writeHead(200, headers);

	const stream = createReadStream(filePath);
	stream.on('error', () => res.destroy());
	stream.pipe(res);
});

server.listen(port, () => {
	console.log(`SPA served from ${root} at http://localhost:${port}`);
});
