---
name: flutter-ui-implementation
description: Implementing mobile UI in Flutter — Material 3 theming, widget composition, responsive layout, animations, and UI performance. Use when building or refactoring Flutter screens, creating reusable widgets, wiring themes, or fixing layout/overflow issues in the hamsa Flutter app.
---

# Flutter UI Implementation

Implementing and refactoring UI in the hamsa Flutter app (mobile/lib, feature-first layout).

## Ground rules

- **Theme, never hardcode**: all colors/text styles via `Theme.of(context)` or the app's design tokens in `lib/core/theme/`. New colors go in the theme first.
- **Composition over custom paint**: prefer existing widgets (`ListTile`, `Card`, `FilledButton`, `SliverAppBar`) before custom widgets. Custom widgets belong in `lib/shared/` once reused.
- **Const everything possible**: `const` constructors on leaf widgets; avoid rebuilding subtrees (scope `Consumer`s/`Selector`s tightly around what depends on state).
- **Riverpod conventions**: UI reads state via `ref.watch`, mutates via controller/notifier methods; screens never call Dio directly.

## Layout

- Design against the narrowest supported width; use `LayoutBuilder`/`MediaQuery` breakpoints, not device-type checks.
- Scrollables: `CustomScrollView` + slivers when mixing app bar/lists; never nest unbounded-height scrollables (watch for `RenderFlex overflow` inside `Column`).
- Text: `TextScaler`-friendly — no fixed heights on text containers; use `maxLines` + `overflow` deliberately.
- Handle keyboard insets (`viewInsets`) so inputs stay visible; use `SafeArea` around edges.

## Motion & feedback

- Durations 150–300 ms, `Curves.easeOutCubic` default; animate only opacity/transform where possible.
- Loading = skeleton or `CircularProgressIndicator.adaptive` with semantic label; buttons disable + show progress during in-flight mutations.
- SnackBar for transient results, dialog only for blocking decisions.

## Verification

- Every UI change is verified on the real surface: run on Chrome (`hub start` per `hamsa-run-dev` skill) and visually confirm the changed screen, including RTL rendering and one error/empty state.
- Widget tests only for observable behavior (state transitions, tap → outcome), not snapshotting.
