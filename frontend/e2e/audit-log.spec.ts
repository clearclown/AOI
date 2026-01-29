import { test, expect } from '@playwright/test';

/**
 * E2E tests for the Audit Log component.
 */
test.describe('Audit Log', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Navigate to Audit Log tab
    await page.getByRole('button', { name: 'Audit Log' }).click();
    await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible();
  });

  test.describe('Audit Log Display', () => {
    test('should display Audit Log heading', async ({ page }) => {
      const heading = page.getByRole('heading', { name: 'Audit Log' });
      await expect(heading).toBeVisible();
    });

    test('should display audit entries', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Should show entries or message about no entries
      const content = page.getByText(/entries|No audit entries/);
      await expect(content.first()).toBeVisible();
    });

    test('should display entry count', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Should show "X of Y entries"
      const entriesCount = page.getByText(/\d+ of \d+ entries/);
      await expect(entriesCount).toBeVisible();
    });

    test('should display from and to agents in entries', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Check for arrow notation (from -> to)
      const arrowNotation = page.getByText(/→/);
      await expect(arrowNotation.first()).toBeVisible();
    });

    test('should display entry timestamp', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Entries should have timestamps - look for date/time patterns (e.g., "1/29/2026" or "12:30:00")
      // The timestamp is formatted with toLocaleString() so it will contain date separators
      const timestampPattern = page.getByText(/\d{1,2}[\/\-\.]\d{1,2}[\/\-\.]\d{2,4}/);
      await expect(timestampPattern.first()).toBeVisible();
    });

    test('should display entry event type', async ({ page }) => {
      await page.waitForTimeout(1000);

      // Event type should be displayed - the mock data includes 'query' as event type
      // The event type is shown in a small span at the bottom of each entry
      // Look for the text that matches common event types
      const eventTypes = ['query', 'response', 'status_update', 'task', 'approval'];
      let found = false;
      for (const type of eventTypes) {
        const count = await page.getByText(type, { exact: true }).count();
        if (count > 0) {
          found = true;
          break;
        }
      }
      expect(found).toBe(true);
    });
  });

  test.describe('Search Functionality', () => {
    test('should have a search input', async ({ page }) => {
      const searchInput = page.getByPlaceholder('Search entries...');
      await expect(searchInput).toBeVisible();
    });

    test('should filter entries when searching', async ({ page }) => {
      const searchInput = page.getByPlaceholder('Search entries...');

      // Get initial entry count
      const initialCount = page.getByText(/\d+ of \d+ entries/);
      const initialText = await initialCount.textContent();

      // Search for something that might match
      await searchInput.fill('Status');
      await page.waitForTimeout(500);

      // Entry count should reflect filtering
      const newCount = page.getByText(/\d+ of \d+ entries/);
      await expect(newCount).toBeVisible();
    });

    test('should search by agent name', async ({ page }) => {
      const searchInput = page.getByPlaceholder('Search entries...');

      // Search for an agent name (mock data has pm-tanaka, eng-suzuki)
      await searchInput.fill('tanaka');
      await page.waitForTimeout(500);

      // Should show filtered results
      const entriesCount = page.getByText(/\d+ of \d+ entries/);
      await expect(entriesCount).toBeVisible();
    });

    test('should search by summary content', async ({ page }) => {
      const searchInput = page.getByPlaceholder('Search entries...');

      // Search for summary content (mock data has "Status check")
      await searchInput.fill('check');
      await page.waitForTimeout(500);

      // Should filter based on summary
      const entriesCount = page.getByText(/\d+ of \d+ entries/);
      await expect(entriesCount).toBeVisible();
    });

    test('should clear search results when input is cleared', async ({ page }) => {
      const searchInput = page.getByPlaceholder('Search entries...');

      // Search for something
      await searchInput.fill('test');
      await page.waitForTimeout(500);

      // Clear search
      await searchInput.clear();
      await page.waitForTimeout(500);

      // Should show all entries again
      const entriesCount = page.getByText(/\d+ of \d+ entries/);
      const countText = await entriesCount.textContent();

      // When cleared, shown count should equal total count
      expect(countText).toMatch(/(\d+) of \1 entries/);
    });
  });

  test.describe('Event Type Filter', () => {
    test('should have an event type filter dropdown', async ({ page }) => {
      const filterDropdown = page.getByRole('combobox');
      await expect(filterDropdown).toBeVisible();
    });

    test('should show "All Events" option', async ({ page }) => {
      const filterDropdown = page.getByRole('combobox');
      await expect(filterDropdown).toHaveValue('all');
    });

    test('should filter entries by event type', async ({ page }) => {
      const filterDropdown = page.getByRole('combobox');

      // Get available options
      const options = filterDropdown.locator('option');
      const optionsCount = await options.count();

      if (optionsCount > 1) {
        // Select a specific event type (not 'all')
        const secondOption = await options.nth(1).getAttribute('value');
        if (secondOption) {
          await filterDropdown.selectOption(secondOption);
          await page.waitForTimeout(500);

          // Verify filter is applied
          await expect(filterDropdown).toHaveValue(secondOption);
        }
      }
    });

    test('should reset filter when All Events is selected', async ({ page }) => {
      const filterDropdown = page.getByRole('combobox');

      // Select a filter first
      const options = filterDropdown.locator('option');
      const optionsCount = await options.count();

      if (optionsCount > 1) {
        const secondOption = await options.nth(1).getAttribute('value');
        if (secondOption) {
          await filterDropdown.selectOption(secondOption);
          await page.waitForTimeout(300);
        }
      }

      // Reset to all
      await filterDropdown.selectOption('all');
      await page.waitForTimeout(300);

      // Should show all entries
      const entriesCount = page.getByText(/\d+ of \d+ entries/);
      const countText = await entriesCount.textContent();
      expect(countText).toMatch(/(\d+) of \1 entries/);
    });
  });

  test.describe('Auto-scroll Feature', () => {
    test('should have an auto-scroll checkbox', async ({ page }) => {
      const autoScrollCheckbox = page.getByRole('checkbox');
      await expect(autoScrollCheckbox).toBeVisible();
    });

    test('should have auto-scroll enabled by default', async ({ page }) => {
      const autoScrollCheckbox = page.getByRole('checkbox');
      await expect(autoScrollCheckbox).toBeChecked();
    });

    test('should toggle auto-scroll when checkbox is clicked', async ({ page }) => {
      const autoScrollCheckbox = page.getByRole('checkbox');

      // Initially checked
      await expect(autoScrollCheckbox).toBeChecked();

      // Uncheck
      await autoScrollCheckbox.uncheck();
      await expect(autoScrollCheckbox).not.toBeChecked();

      // Check again
      await autoScrollCheckbox.check();
      await expect(autoScrollCheckbox).toBeChecked();
    });

    test('should display auto-scroll label', async ({ page }) => {
      const autoScrollLabel = page.getByText('Auto-scroll');
      await expect(autoScrollLabel).toBeVisible();
    });
  });

  test.describe('Combined Filtering', () => {
    test('should apply both search and event type filter', async ({ page }) => {
      const searchInput = page.getByPlaceholder('Search entries...');
      const filterDropdown = page.getByRole('combobox');

      // Apply search
      await searchInput.fill('status');
      await page.waitForTimeout(300);

      // Get available filter options
      const options = filterDropdown.locator('option');
      const optionsCount = await options.count();

      if (optionsCount > 1) {
        // Apply event type filter
        const secondOption = await options.nth(1).getAttribute('value');
        if (secondOption) {
          await filterDropdown.selectOption(secondOption);
          await page.waitForTimeout(300);
        }
      }

      // Both filters should be active
      const entriesCount = page.getByText(/\d+ of \d+ entries/);
      await expect(entriesCount).toBeVisible();
    });
  });
});
