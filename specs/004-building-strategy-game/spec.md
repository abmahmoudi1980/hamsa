# Spec: Building Strategy Game

## Objective

Add an optional, single-player building-management strategy game to Hamsa's web
manager console. It is a fictional sandbox for entertainment: it must not read,
write, or affect real building, resident, payment, or maintenance records.

The player manages a fictional apartment building for a 12-month campaign. Each
month, they choose how to spend a limited pool of fictional credits on upkeep
or upgrades, then respond to a game event. Their choices affect building
condition, resident satisfaction, and operating efficiency. The campaign ends
with a score and outcome based on those measures.

### MVP acceptance criteria

- An authenticated manager can open `/m/game`, start a campaign, and resume a
  saved campaign.
- The game presents the current month, fictional credit balance, building
  condition, resident satisfaction, efficiency, available actions, and recent
  events in Persian and RTL.
- The player can select an affordable action and advance the month. An
  unaffordable action cannot be applied.
- A 12-month campaign ends with a summary and a way to start over.
- Campaign state persists in browser local storage and can be reset by the
  player.
- The interface is usable at mobile and desktop widths, with keyboard
  operability and visible focus states.
- Game state has no backend persistence and causes no requests that mutate
  Hamsa business data.

## Tech Stack

- Svelte 5 / SvelteKit 3 SPA, TypeScript 6 strict mode, Tailwind CSS 4.
- Existing Hamsa design tokens, Persian-only localization, and RTL layout.
- Existing Vitest and Playwright tooling; no new dependencies.
- Game rules are pure client-side TypeScript. Browser local storage is used for
  save/resume.

## Commands

Run from `web/`:

```sh
npm run dev
npm run check
npm run test:unit -- --run
npm run test:e2e
npm run build
```

## Project Structure

```text
web/src/routes/m/game/+page.svelte       Game route and interaction
web/src/lib/game/                        Game model, rules, persistence, tests
web/src/i18n/fa.ts                        Persian game labels
specs/004-building-strategy-game/        Feature spec, plan, and task list
```

## Code Style

Follow the current Svelte 5 runes, strict TypeScript, and existing component
patterns. Keep game rules independent of UI so they can be tested without a
browser. Example interaction:

```svelte
<button type="button" disabled={!canAfford(action)} onclick={() => chooseAction(action)}>
  {action.label}
</button>
```

Do not add source-code comments.

## Testing Strategy

- Unit-test campaign creation, action affordability/effects, month progression,
  event resolution, completion scoring, and save-data validation using Vitest.
- Test persistence round-trips and malformed saved data safely falls back to a
  new campaign.
- Add a Playwright flow covering start, action selection, month advancement,
  reload/resume, campaign completion, and reset.
- Run `npm run check`, unit tests, and `npm run build`; run the focused E2E flow
  when the local test environment supports it.

## Boundaries

- Always: keep the game fictional, client-side, Persian RTL, and isolated from
  real Hamsa business data; validate persisted state; maintain responsive and
  accessible interactions.
- Ask first: adding dependencies, backend endpoints or database changes,
  multiplayer/leaderboards, cross-client support, or use of real building data.
- Never: modify real balances, invoices, maintenance requests, residents, or
  other production records as a game effect.

## Success Criteria

The acceptance criteria above pass; a manager can complete the 12-month game,
make strategic choices that affect the final outcome, reload without losing an
in-progress campaign, and reset it without any backend mutation.

## Decisions Confirmed

- Audience: regular building managers only, with a link in the manager console.
- Game balance, action/event choices, and scoring are defined in the approved
  implementation plan.
