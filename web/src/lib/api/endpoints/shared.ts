/** Standard pagination envelope (contracts/api.md §Conventions). */
export interface Page<T> {
	items: T[];
	page: number;
	page_size: number;
	total: number;
}
