# Implementation Plan: Building Strategy Game

**Branch**: `004-building-strategy-game` | **Spec**: [spec.md](spec.md)

## Summary

Add a client-only, 12-month building strategy campaign to the authenticated
manager console at `/m/game`. Keep campaign rules in pure TypeScript, persist a
versioned save in local storage, and keep all game data separate from Hamsa's
API and business records. The interface uses the existing Persian RTL design
system and is linked from both desktop and mobile manager navigation.

## Technical Context

- **Client**: Svelte 5 / SvelteKit 3 SPA, TypeScript 6 strict mode, Tailwind CSS 4.
- **Existing support**: `zod` for save validation, Vitest for unit tests,
  Playwright for E2E, and `web/src/i18n/fa.ts` as the Persian UI text source.
- **Persistence**: versioned browser local storage, read and written only in
  browser lifecycle code. No API, migration, or dependency changes.
- **Role access**: regular building managers only. The manager shell also
  admits superadmins, so the game page must reject direct superadmin access as
  well as omit the nav link for that role.
- **Campaign**: one action plus one event decision per month for 12 months.
  Fictional credits and three bounded metrics (condition, satisfaction, and
  efficiency) feed the final score. Events are selected deterministically from
  saved campaign state so reloads cannot reroll an outcome.

### Proposed first-pass balance

- Start at 55 credits; grant 35 at the start of each month.
- Choose one building action per month. Initial candidates: preventive
  maintenance, insulation, community space, and elevator modernization, each
  with a distinct cost/effect profile.
- Resolve one event per month by choosing one of two responses, typically a
  credit cost versus a metric tradeoff. Include a safe no-cost choice where
  needed to avoid a forced negative balance.
- Clamp metrics to 0–100. Score the final campaign from the three metrics;
  provide a clear outcome tier and a breakdown of the final score.
- Keep costs and effects in a single typed game configuration so balance can be
  adjusted without changing UI code.

## Project Structure

```text
web/src/lib/game/model.ts                 Types, initial state, constants
web/src/lib/game/engine.ts                Pure turn/action/event/score rules
web/src/lib/game/persistence.ts           Versioned save validation and storage
web/src/lib/game/engine.test.ts           Game-rule tests
web/src/lib/game/persistence.test.ts      Save/load and invalid-data tests
web/src/routes/m/game/+page.svelte        Game UI and browser-only lifecycle
web/src/routes/m/+layout.svelte           Manager nav links (desktop and mobile)
web/src/i18n/fa.ts                        Persian game labels and event copy
web/e2e/game.e2e.ts                       Authenticated manager gameplay flow
```

Adjust exact file split to match existing conventions during implementation;
keep rule logic testable without importing Svelte or browser globals.

## Implementation Sequence

1. **Game model and rules** — define typed state/configuration, initial
   campaign, action and event choices, bounded metric updates, deterministic
   event selection, month transitions, completion, and scoring. Verify with
   focused engine tests.
2. **Save/resume** — add a versioned storage envelope and schema validation;
   safely handle missing, malformed, or unsupported saves. Verify round-trip,
   reset, and corrupted-save behavior.
3. **Game screen** — implement intro/new campaign, in-progress campaign,
   action/event choices, month progression, final summary, resume, and reset.
   Access local storage only after browser mount. Check Persian text, RTL,
   narrow viewport layout, keyboard controls, and focus visibility.
4. **Console integration** — add the manager-only `/m/game` navigation entry
   in both responsive navs, add Persian strings, and redirect non-manager roles
   away from the route. No API calls or backend changes.
5. **End-to-end verification** — add the manager `auth.setup.ts` fixture
   expected by the existing Playwright config, using explicitly configured
   test-manager credentials to sign in and save browser storage state. Then
   cover start, an affordable choice, rejected unaffordable choice, progression,
   reload/resume, completion, and reset.

## Verification Checkpoints

Run from `web/`:

```sh
npm run check
npm run test:unit -- --run
npx playwright test e2e/game.e2e.ts
npm run build
```

The E2E setup requires a reachable real backend, a valid test-manager account,
and the environment variables documented by the fixture. If those prerequisites
are unavailable, report the limitation and still run the type check, unit tests,
and production build.

## Risks and Mitigations

- **Save data corruption or stale formats**: validate a versioned envelope and
  discard invalid game-only data without affecting app session or API data.
- **Static rendering/browser API mismatch**: defer local storage access until
  the component mounts in a browser.
- **Randomness makes saves or tests inconsistent**: store deterministic random
  state with the campaign and test fixed states.
- **Strategy feels solved or punishing**: centralize costs/effects, include
  tradeoffs, and play through multiple test campaigns before finalizing score
  bands.
- **Superadmin shares the manager shell**: enforce role at the route and at
  navigation visibility, not just through the shell.
- **Mobile navigation is crowded**: use the existing horizontally scrollable
  nav and verify the game link remains reachable at narrow widths.
- **Playwright auth bootstrap is not implemented yet**: the config references
  `e2e/auth.setup.ts`, but the `web/e2e/` directory is currently absent. Add a
  fixture that reads credentials only from environment variables; never commit
  test credentials or an authenticated state file.

## Boundaries

- No backend endpoints, database changes, external game assets, dependencies,
  real building data, multiplayer, or leaderboard in this MVP.
- Do not reuse real resident identities, balances, expenses, invoices, or
  maintenance records as game state.
- Do not write campaign state to any storage other than the versioned,
  game-specific local storage key.

## Open Decision for Review

The route, audience, sandbox boundary, 12-month loop, and first-pass campaign
balance follow the approved spec and plan. The missing E2E setup has been added
to the plan based on the current repository state.
