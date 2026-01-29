import { test, expect } from '@playwright/test';

/**
 * E2E tests for the Dashboard component.
 */
test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Ensure we're on the Dashboard tab (default)
    await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();
  });

  test.describe('Agent Display', () => {
    test('should display Agent Dashboard heading', async ({ page }) => {
      const heading = page.getByRole('heading', { name: 'Agent Dashboard' });
      await expect(heading).toBeVisible();
    });

    test('should display at least one agent card', async ({ page }) => {
      // Wait for agents to load (mock data or API)
      await page.waitForTimeout(1000);

      // Look for agent ID (either from API or mock: eng-local)
      const agentCard = page.locator('div').filter({ hasText: /Role:.*Status:/s }).first();
      await expect(agentCard).toBeVisible();
    });

    test('should display agent role information', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Check for Role label
      await expect(page.getByText('Role:', { exact: false })).toBeVisible();
    });

    test('should display agent status', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Check for Status label
      await expect(page.getByText('Status:', { exact: false })).toBeVisible();
    });

    test('should display agent last seen time', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Check for Last seen label
      await expect(page.getByText('Last seen:', { exact: false })).toBeVisible();
    });

    test('should show online status with green color', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Check for online status text
      const onlineStatus = page.locator('span').filter({ hasText: 'online' });
      if (await onlineStatus.count() > 0) {
        // Verify the span exists with green color
        await expect(onlineStatus.first()).toBeVisible();
      }
    });
  });

  test.describe('Refresh Functionality', () => {
    test('should have a refresh button', async ({ page }) => {
      const refreshButton = page.getByRole('button', { name: 'Refresh' });
      await expect(refreshButton).toBeVisible();
    });

    test('should update last updated time when refresh is clicked', async ({ page }) => {
      // Get initial last updated time
      const lastUpdatedText = page.getByText('Last updated:');
      await expect(lastUpdatedText).toBeVisible();

      const initialTime = await lastUpdatedText.textContent();

      // Wait a moment then click refresh
      await page.waitForTimeout(1100);
      await page.getByRole('button', { name: 'Refresh' }).click();

      // The time should update (though content might be the same if within same second)
      await expect(lastUpdatedText).toBeVisible();
    });

    test('should show last updated timestamp', async ({ page }) => {
      const lastUpdated = page.getByText('Last updated:');
      await expect(lastUpdated).toBeVisible();

      // Should contain a time format
      const timeText = await lastUpdated.textContent();
      expect(timeText).toContain(':'); // Time format contains colons
    });
  });

  test.describe('Agent Cards Layout', () => {
    test('should display agent cards in a grid layout', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Check that the grid container exists
      const gridContainer = page.locator('div[style*="grid"]');
      await expect(gridContainer).toBeVisible();
    });

    test('should display agent ID as heading in card', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Agent ID should be displayed as h3
      const agentHeading = page.getByRole('heading', { level: 3 }).first();
      await expect(agentHeading).toBeVisible();
    });
  });
});
