import { QueryClient } from '@tanstack/svelte-query';
import { ApiError } from '#lib/api/apiError';

/**
 * Query defaults.
 *
 * The app is a management console over data that changes when the manager
 * acts, so caching is tuned to keep screens instant while still refetching
 * after a mutation:
 *   - `staleTime` short: a manager who records a payment in one tab expects the
 *     ledger in another tab to be current, and 15s is a good balance against
 *     re-fetching on every focus change.
 *   - Retries only on transient faults. A 4xx (validation, forbidden, conflict)
 *     will fail identically on retry, so retrying it only wastes requests and
 *     delays the error the user needs to see.
 */
export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			staleTime: 15_000,
			gcTime: 5 * 60_000,
			refetchOnWindowFocus: true,
			retry(failureCount, error) {
				if (error instanceof ApiError) {
					if (!error.isTransient) return false;
					// Rate limiting needs a longer pause than a plain network blip.
					return error.status !== 429 ? failureCount < 2 : failureCount < 1;
				}
				return failureCount < 2;
			}
		},
		mutations: {
			// Never auto-retry a mutation: it may not be idempotent (issuing
			// invoices, recording a payment), and the server guards with 409s.
			retry: false
		}
	}
});
