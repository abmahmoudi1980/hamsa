import { mkdir } from 'node:fs/promises';
import { dirname } from 'node:path';
import { expect, test as setup } from '@playwright/test';

const authFile = 'e2e/.auth/manager.json';

setup('authenticate manager', async ({ page, context }) => {
	await mkdir(dirname(authFile), { recursive: true });

	const phone = process.env.HAMSA_E2E_MANAGER_PHONE;
	const password = process.env.HAMSA_E2E_MANAGER_PASSWORD;
	if (!phone || !password) {
		await context.storageState({ path: authFile });
		return;
	}

	await page.goto('/login');
	await page.getByLabel('شماره موبایل').fill(phone);
	await page.getByLabel('رمز عبور').fill(password);
	await page.getByRole('button', { name: 'ورود', exact: true }).click();
	await expect(page).toHaveURL(/\/m(?:$|\/)/);
	await context.storageState({ path: authFile });
});
