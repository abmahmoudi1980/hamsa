# Tasks: Building Strategy Game

Each task is a focused implementation slice with an explicit verification
command. Run commands from `web/` unless otherwise stated.

- [x] **Task 1: Implement deterministic campaign rules**
  - Acceptance: typed initial state, fictional actions/events, bounded metrics,
    affordability checks, month progression, seeded event selection, 12-month
    completion, and score summary are implemented as pure functions.
  - Verify: `npm run test:unit -- --run src/lib/game/engine.test.ts` and
    `npm run check`.
  - Files: `src/lib/game/model.ts`, `src/lib/game/engine.ts`,
    `src/lib/game/engine.test.ts`.

- [x] **Task 2: Add validated campaign save/resume**
  - Acceptance: a versioned save can round-trip through an injected storage
    adapter; reset removes only the game save; malformed or unsupported data
    is rejected safely without throwing or touching app session storage.
  - Verify: `npm run test:unit -- --run src/lib/game/persistence.test.ts` and
    `npm run check`.
  - Files: `src/lib/game/persistence.ts`,
    `src/lib/game/persistence.test.ts`.

- [x] **Task 3: Build the Persian game screen**
  - Acceptance: `/m/game` supports intro, new/resumed campaign, action and event
    choice, month advancement, completed summary, and reset. Storage is accessed
    only in browser lifecycle code. Direct superadmin access redirects away.
    All game copy is Persian, with RTL layout, responsive controls, visible
    focus, and affordable-choice feedback.
  - Verify: `npm run check`, `npm run test:unit -- --run`, and `npm run build`.
  - Files: `src/routes/m/game/+page.svelte`, `src/i18n/fa.ts`.

- [x] **Task 4: Add manager-console navigation**
  - Acceptance: a game link is visible in desktop and mobile navigation only to
    regular managers, points to `/m/game`, and fits the existing responsive
    navigation behavior.
  - Verify: `npm run check` and `npm run build`.
  - Files: `src/routes/m/+layout.svelte`.

- [x] **Task 5: Add authenticated Playwright coverage**
  - Acceptance: `auth.setup.ts` signs in with credentials supplied only by
    environment variables and writes the configured storage state; the game
    flow test covers start, action/event selection, affordability, progression,
    reload/resume, completion, and reset. No secrets or storage-state artifacts
    are committed.
  - Verify: with the real backend available and a test-manager account
    configured, run `npx playwright test e2e/game.e2e.ts`; otherwise document
    the missing prerequisite and run the other checks.
  - Files: `e2e/auth.setup.ts`, `e2e/game.e2e.ts`, `.gitignore`.
  - Verification note: Playwright test discovery passed. The end-to-end flow
    could not run because no manager test credentials are configured.
