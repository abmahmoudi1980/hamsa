import { expect, test } from '@playwright/test';

test('manager can play, save, resume, finish, and reset a campaign', async ({ page }) => {
	if (!process.env.HAMSA_E2E_MANAGER_PHONE || !process.env.HAMSA_E2E_MANAGER_PASSWORD) {
		test.skip(
			true,
			'Set HAMSA_E2E_MANAGER_PHONE and HAMSA_E2E_MANAGER_PASSWORD for real-backend E2E.'
		);
	}

	await page.goto('/m/game');
	await page.getByRole('button', { name: 'شروع بازی', exact: true }).click();
	await expect(page.getByTestId('game-action-maintenance')).toBeEnabled();

	await page.evaluate(() => {
		const key = Object.keys(localStorage).find((value) =>
			value.startsWith('hamsa.strategy-game.v1:')
		);
		if (!key) throw new Error('Game save was not written.');
		const envelope = JSON.parse(localStorage.getItem(key) ?? 'null');
		envelope.campaign.credits = 10;
		localStorage.setItem(key, JSON.stringify(envelope));
	});
	await page.reload();
	await page.getByRole('button', { name: 'ادامه بازی ذخیره‌شده', exact: true }).click();
	await expect(page.getByTestId('game-action-maintenance')).toBeDisabled();
	await expect(page.getByText('اعتبار کافی نیست').first()).toBeVisible();

	await page.getByRole('button', { name: 'شروع دوباره', exact: true }).click();
	await page.getByRole('button', { name: 'بله، از نو شروع کن', exact: true }).click();
	await page.getByTestId('game-action-maintenance').click();
	await page.locator('[data-testid^="game-choice-"]:not([disabled])').last().click();

	await page.reload();
	await page.getByRole('button', { name: 'ادامه بازی ذخیره‌شده', exact: true }).click();
	await expect(page.getByTestId('game-action-maintenance')).toBeEnabled();

	for (let month = 2; month <= 12; month += 1) {
		await page.getByTestId('game-action-maintenance').click();
		await page.locator('[data-testid^="game-choice-"]:not([disabled])').last().click();
	}

	await expect(page.getByTestId('game-score')).toBeVisible();
	await page.getByRole('button', { name: 'شروع بازی تازه', exact: true }).click();
	await page.getByRole('button', { name: 'بله، از نو شروع کن', exact: true }).click();
	await expect(page.getByTestId('game-action-maintenance')).toBeEnabled();
});
