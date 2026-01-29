import { test as base, expect } from '@playwright/test';

/**
 * Custom test fixtures for AOI Protocol E2E tests.
 * Provides common setup and utilities for all tests.
 */

// Extend base test with custom fixtures
export const test = base.extend<{
  // Custom fixture for app loaded state
  appLoaded: void;
}>({
  appLoaded: async ({ page }, use) => {
    // Navigate to the app and wait for it to load
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'AOI Protocol Dashboard' })).toBeVisible();
    await use();
  },
});

export { expect };

/**
 * Wait for connection status indicator to appear
 */
export async function waitForConnectionStatus(page: import('@playwright/test').Page) {
  // Wait for the connection status text (either Connected, Disconnected, or Checking)
  await expect(page.getByText(/Connected|Disconnected|Checking/)).toBeVisible({ timeout: 10000 });
}

/**
 * Navigate to a specific tab
 */
export async function navigateToTab(page: import('@playwright/test').Page, tabName: 'Dashboard' | 'Audit Log' | 'Approvals') {
  await page.getByRole('button', { name: tabName }).click();
}

/**
 * Check if using mock data (disconnected from backend)
 */
export async function isUsingMockData(page: import('@playwright/test').Page): Promise<boolean> {
  const connectionText = await page.getByText(/Connected|Disconnected/).textContent();
  return connectionText?.includes('Disconnected') ?? false;
}
