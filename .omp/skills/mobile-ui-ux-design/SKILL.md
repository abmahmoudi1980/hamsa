---
name: mobile-ui-ux-design
description: Mobile application UI/UX design review and guidance — layout hierarchy, typography, color systems, touch ergonomics, accessibility, and UX flows. Use when designing new screens, reviewing app UI, creating design specs, or improving user experience of mobile apps (Flutter, Android, iOS).
---

# Mobile UI/UX Design

Apply this when designing or reviewing mobile app screens and flows.

## Workflow

1. **Clarify the job**: who is the user (role), what task, on what device. One primary action per screen; everything else is secondary or overflow.
2. **Map the flow before pixels**: entry point → decision points → success state → error states → empty states. A screen without a designed empty/error state is incomplete.
3. **Design with tokens, not magic numbers**: spacing scale (4/8/12/16/24/32/48), type scale (display/headline/title/body/label), semantic color roles (primary, surface, error, success) — never raw hex/px sprinkled through code.
4. **Verify ergonomics on the smallest target device**: primary actions reachable in the bottom third; touch targets ≥ 48×48 dp; destructive actions away from frequent taps or confirmed.
5. **Accessibility pass**: text contrast ≥ 4.5:1 (3:1 for large text), no color-only state signaling, dynamic type support (no fixed-height text containers), meaningful semantic labels for screen readers.

## Review checklist

- Visual hierarchy: can a first-time user find the primary action in < 3 s?
- State coverage: loading (skeleton, not spinner-void), empty (with action), error (with recovery), success.
- Feedback: every tap gives a perceptible response < 100 ms; long operations show progress.
- Navigation: back behavior consistent; deep flows show where you are (title/progress); no dead ends.
- Consistency: same action looks the same everywhere; platform conventions respected (Material on Android, HIG on iOS) unless brand mandates otherwise.
- Content: real copy, not lorem; numbers/dates localized; truncation rules defined for long text.

## UX principles

- Recognition over recall: show options, don't force memorization.
- Progressive disclosure: advanced/low-frequency settings behind secondary entry.
- Forgiving design: undo over confirm where possible; confirm only for irreversible/destructive.
- Optimize for the 80% path; power-user shortcuts must not clutter the primary path.

## Deliverables

When producing design output, state: target users, screen inventory, flow diagram (mermaid if useful), component/token list, and open questions. When reviewing, output a prioritized list (blocker / major / minor / nit) with concrete fixes, not vague praise.
