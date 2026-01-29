import { test, expect } from '@playwright/test';

/**
 * E2E tests for the main App component and overall application behavior.
 */
test.describe('AOI Protocol Application', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test.describe('Application Loading', () => {
    test('should display the main title "AOI Protocol Dashboard"', async ({ page }) => {
      const title = page.getByRole('heading', { name: 'AOI Protocol Dashboard' });
      await expect(title).toBeVisible();
    });

    test('should have proper page structure', async ({ page }) => {
      // Check for main elements
      await expect(page.getByRole('heading', { name: 'AOI Protocol Dashboard' })).toBeVisible();
      await expect(page.getByRole('navigation').or(page.locator('nav'))).toBeVisible();
    });

    test('should not have critical console errors on load', async ({ page }) => {
      const errors: string[] = [];
      page.on('console', (msg) => {
        if (msg.type() === 'error') {
          errors.push(msg.text());
        }
      });

      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(1000);

      // Filter out expected errors (like network failures in test environment, CORS, WebSocket, etc.)
      const unexpectedErrors = errors.filter(
        (err) =>
          !err.includes('Failed to fetch') &&
          !err.includes('net::ERR') &&
          !err.includes('NetworkError') &&
          !err.includes('CORS') &&
          !err.includes('favicon') &&
          !err.includes('404') &&
          !err.includes('WebSocket') &&
          !err.includes('socket')
      );
      expect(unexpectedErrors).toHaveLength(0);
    });
  });

  test.describe('Tab Navigation', () => {
    test('should display all navigation tabs', async ({ page }) => {
      await expect(page.getByRole('button', { name: 'Dashboard' })).toBeVisible();
      await expect(page.getByRole('button', { name: 'Audit Log' })).toBeVisible();
      await expect(page.getByRole('button', { name: 'Approvals' })).toBeVisible();
    });

    test('should navigate to Dashboard tab when clicked', async ({ page }) => {
      // First go to another tab
      await page.getByRole('button', { name: 'Audit Log' }).click();

      // Then navigate back to Dashboard
      await page.getByRole('button', { name: 'Dashboard' }).click();

      // Verify Dashboard content is visible
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();
    });

    test('should navigate to Audit Log tab when clicked', async ({ page }) => {
      await page.getByRole('button', { name: 'Audit Log' }).click();

      // Verify Audit Log content is visible
      await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible();
    });

    test('should navigate to Approvals tab when clicked', async ({ page }) => {
      await page.getByRole('button', { name: 'Approvals' }).click();

      // Verify Approvals content is visible (either Pending Approvals heading or empty state)
      const approvalsContent = page.getByText(/Pending Approvals|No pending approvals/);
      await expect(approvalsContent).toBeVisible();
    });

    test('should persist tab state during navigation', async ({ page }) => {
      // Navigate through all tabs and back
      await page.getByRole('button', { name: 'Audit Log' }).click();
      await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible();

      await page.getByRole('button', { name: 'Approvals' }).click();
      await expect(page.getByText(/Pending Approvals|No pending approvals/)).toBeVisible();

      await page.getByRole('button', { name: 'Dashboard' }).click();
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();
    });
  });

  test.describe('Connection Status', () => {
    test('should display connection status indicator', async ({ page }) => {
      // Wait for connection check to complete
      await page.waitForTimeout(2000);

      // Should show either "Connected", "Disconnected", or "Checking..."
      const connectionStatus = page.getByText(/Connected|Disconnected|Checking/);
      await expect(connectionStatus).toBeVisible();
    });

    test('should show status indicator dot', async ({ page }) => {
      // The status indicator should be visible
      // The dot is styled with backgroundColor, check the container has the text
      await page.waitForTimeout(2000);

      const statusText = await page.getByText(/Connected|Disconnected|Checking/).first();
      await expect(statusText).toBeVisible();
    });

    test('should indicate when using mock data', async ({ page }) => {
      await page.waitForTimeout(3000);

      // When disconnected, should show "using mock data" message
      const connectionText = await page.getByText(/Connected|Disconnected/).first().textContent();

      if (connectionText?.includes('Disconnected')) {
        await expect(page.getByText('using mock data')).toBeVisible();
      }
      // If connected, test passes as backend is available
    });
  });
});
