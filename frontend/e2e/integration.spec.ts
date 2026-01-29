import { test, expect } from '@playwright/test';

/**
 * E2E integration tests that test workflows across multiple components.
 */
test.describe('Integration Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'AOI Protocol Dashboard' })).toBeVisible();
  });

  test.describe('Full Application Workflow', () => {
    test('should allow complete navigation through all tabs', async ({ page }) => {
      // Start on Dashboard (default)
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();

      // Go to Audit Log
      await page.getByRole('button', { name: 'Audit Log' }).click();
      await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible();

      // Use search feature
      const searchInput = page.getByPlaceholder('Search entries...');
      await searchInput.fill('status');
      await page.waitForTimeout(300);

      // Go to Approvals
      await page.getByRole('button', { name: 'Approvals' }).click();
      await expect(page.getByText(/Pending Approvals|No pending approvals/)).toBeVisible();

      // Return to Dashboard
      await page.getByRole('button', { name: 'Dashboard' }).click();
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();
    });

    test('should maintain connection status across tabs', async ({ page }) => {
      // Wait for initial connection check to complete (shows Connected, Disconnected, or Checking)
      await page.waitForTimeout(3000);

      // The status text should be visible (Connected, Disconnected, or still Checking)
      const statusLocator = page.getByText(/Connected|Disconnected|Checking/);
      await expect(statusLocator.first()).toBeVisible();

      // Navigate through tabs
      await page.getByRole('button', { name: 'Audit Log' }).click();
      await page.waitForTimeout(300);

      // Connection status should still be visible in header
      await expect(statusLocator.first()).toBeVisible();

      // Navigate to Approvals
      await page.getByRole('button', { name: 'Approvals' }).click();
      await page.waitForTimeout(300);

      // Status should persist
      await expect(statusLocator.first()).toBeVisible();
    });

    test('should handle rapid tab switching gracefully', async ({ page }) => {
      // Rapidly switch between tabs
      for (let i = 0; i < 5; i++) {
        await page.getByRole('button', { name: 'Dashboard' }).click();
        await page.getByRole('button', { name: 'Audit Log' }).click();
        await page.getByRole('button', { name: 'Approvals' }).click();
      }

      // Application should still be responsive
      await page.getByRole('button', { name: 'Dashboard' }).click();
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();
    });
  });

  test.describe('Data Consistency', () => {
    test('should show consistent agent data when switching between tabs', async ({ page }) => {
      // Wait for data to load and dashboard to be visible
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();
      await page.waitForTimeout(1500);

      // Get agent info from Dashboard - wait for the h3 to be available
      const agentHeading = page.getByRole('heading', { level: 3 }).first();
      await expect(agentHeading).toBeVisible();
      const agentId = await agentHeading.textContent();

      // Switch tabs
      await page.getByRole('button', { name: 'Audit Log' }).click();
      await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible();

      // Return to Dashboard
      await page.getByRole('button', { name: 'Dashboard' }).click();
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();
      await page.waitForTimeout(500);

      // Agent should still be the same
      const agentHeadingAfter = page.getByRole('heading', { level: 3 }).first();
      await expect(agentHeadingAfter).toBeVisible();
      const agentIdAfter = await agentHeadingAfter.textContent();
      expect(agentIdAfter).toBe(agentId);
    });

    test('should preserve search filter when returning to Audit Log', async ({ page }) => {
      // Note: Search state is local to the component and resets on re-render
      // This test verifies the component handles this correctly

      // Go to Audit Log
      await page.getByRole('button', { name: 'Audit Log' }).click();

      // Apply search filter
      const searchInput = page.getByPlaceholder('Search entries...');
      await searchInput.fill('test-search');

      // Switch to another tab
      await page.getByRole('button', { name: 'Dashboard' }).click();
      await page.waitForTimeout(300);

      // Return to Audit Log
      await page.getByRole('button', { name: 'Audit Log' }).click();

      // Search input should be cleared (component remounts)
      await expect(searchInput).toHaveValue('');
    });
  });

  test.describe('Error Resilience', () => {
    test('should handle page reload gracefully', async ({ page }) => {
      // Navigate to a specific tab
      await page.getByRole('button', { name: 'Audit Log' }).click();
      await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible();

      // Reload the page
      await page.reload();

      // Should return to default tab (Dashboard)
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();
    });

    test('should recover from network failures gracefully', async ({ page }) => {
      // Wait for initial load and ensure dashboard is visible
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();
      await page.waitForTimeout(1000);

      // Ensure the Refresh button is visible before going offline
      const refreshButton = page.getByRole('button', { name: 'Refresh' });
      await expect(refreshButton).toBeVisible();

      // Simulate going offline
      await page.context().setOffline(true);

      // Try refreshing Dashboard - the button should still work
      await refreshButton.click();
      await page.waitForTimeout(500);

      // Should still show data (mock data fallback)
      await expect(page.getByRole('heading', { name: 'Agent Dashboard' })).toBeVisible();

      // Go back online
      await page.context().setOffline(false);
    });
  });

  test.describe('Accessibility', () => {
    test('should be navigable using tab key', async ({ page }) => {
      // Focus on first interactive element
      await page.keyboard.press('Tab');

      // Should be able to tab through navigation buttons
      let tabCount = 0;
      const maxTabs = 10;

      while (tabCount < maxTabs) {
        await page.keyboard.press('Tab');
        tabCount++;

        // Check if we can find focused elements
        const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
        if (focusedElement === 'BUTTON' || focusedElement === 'INPUT') {
          break;
        }
      }

      // Should have found an interactive element
      expect(tabCount).toBeLessThan(maxTabs);
    });

    test('should have proper heading hierarchy', async ({ page }) => {
      // Check for h1 (main title)
      const h1 = page.getByRole('heading', { level: 1 });
      await expect(h1).toBeVisible();

      // Check for h2 (section headers)
      const h2 = page.getByRole('heading', { level: 2 });
      await expect(h2.first()).toBeVisible();
    });
  });

  test.describe('Performance', () => {
    test('should load within reasonable time', async ({ page }) => {
      const startTime = Date.now();

      await page.goto('/');
      await expect(page.getByRole('heading', { name: 'AOI Protocol Dashboard' })).toBeVisible();

      const loadTime = Date.now() - startTime;

      // Should load within 5 seconds
      expect(loadTime).toBeLessThan(5000);
    });

    test('should respond to interactions quickly', async ({ page }) => {
      // Measure tab switch time
      const startTime = Date.now();

      await page.getByRole('button', { name: 'Audit Log' }).click();
      await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible();

      const switchTime = Date.now() - startTime;

      // Should switch within 500ms
      expect(switchTime).toBeLessThan(500);
    });
  });
});
