import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for AOI Protocol E2E tests.
 *
 * Usage:
 * - Local dev: `npm run test:e2e` (starts Vite dev server automatically)
 * - With UI: `npm run test:e2e:ui` (opens Playwright UI mode)
 * - Headed: `npm run test:e2e:headed` (shows browser window)
 * - Docker: `npm run test:e2e:docker` (uses Docker Compose)
 *
 * @see https://playwright.dev/docs/test-configuration
 */

const useDocker = process.env.USE_DOCKER === 'true' || process.env.E2E_BASE_URL?.includes('aoi-frontend');
const baseURL = process.env.E2E_BASE_URL || (useDocker ? 'http://aoi-frontend' : 'http://localhost:5173');

export default defineConfig({
  testDir: './e2e',

  /* Run tests in files in parallel */
  fullyParallel: true,

  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,

  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,

  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 1 : undefined,

  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list']
  ],

  /* Global test timeout */
  timeout: 30 * 1000,

  /* Expect timeout */
  expect: {
    timeout: 5000,
  },

  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL,

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'on-first-retry',

    /* Screenshot on failure */
    screenshot: 'only-on-failure',

    /* Video on failure */
    video: 'on-first-retry',

    /* Viewport size */
    viewport: { width: 1280, height: 720 },
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    // Uncomment to test in more browsers
    // {
    //   name: 'firefox',
    //   use: { ...devices['Desktop Firefox'] },
    // },
    // {
    //   name: 'webkit',
    //   use: { ...devices['Desktop Safari'] },
    // },
  ],

  /* Run your local dev server before starting the tests */
  webServer: process.env.E2E_BASE_URL ? undefined : (useDocker ? {
    // When using Docker Compose
    command: 'cd .. && docker compose up',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
    stdout: 'pipe',
    stderr: 'pipe',
  } : {
    // When using local Vite dev server
    command: 'npm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
    timeout: 60 * 1000,
    stdout: 'pipe',
    stderr: 'pipe',
  }),

  /* Output folder for test artifacts */
  outputDir: 'test-results',
});
