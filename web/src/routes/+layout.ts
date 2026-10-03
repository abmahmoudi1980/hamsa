/**
 * Root layout module.
 *
 * `ssr = false` selects SvelteKit's SPA mode: the app is entirely behind a
 * login, so there is no public page worth server-rendering, and the static
 * adapter writes a single index.html that client-side routing takes over from.
 * Combined with `adapter({ fallback: 'index.html' })` in vite.config.ts this
 * produces a deployable bundle of static files.
 *
 * `prerender = false` keeps the adapter from attempting to crawl routes, which
 * would fail for the same reason: every route needs a session.
 */
export const ssr = false;
export const prerender = false;

/** Trailing slashes off, matching the API's flat paths. */
export const trailingSlash = 'never';
