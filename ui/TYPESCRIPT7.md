# TypeScript 7 in the UI

## Current setup (2026-09-06)

- `typescript` is real **TypeScript 7.0.2** (no `@typescript/typescript6` alias).
- `type-check` runs `tsc` from that package.
- Lint runs **oxlint** (+ `oxlint-tsgolint` for optional type-aware rules later). ESLint + `typescript-eslint` were removed because they cannot consume TS 7.0.
- `check:mui9` still needs a programmatic TS AST (`createSourceFile`). TS 7.0 has no JS parser API, so that script imports **`@typescript/typescript6`** (tooling-only; type-check stays on real TS 7).

## Why not typescript-eslint?

TypeScript 7.0 ships **no programmatic JS compiler API**. `typescript-eslint` (including canary `8.69.1-alpha.0`) still peers `typescript >=4.8.4 <6.1.0` and hard-rejects 7.0 at runtime (`typescript-eslint#12518` closed as blocked; tracking `typescript-eslint#10940`).

Microsoft's documented workaround is the dual alias (`typescript` = `@typescript/typescript6`, native checker via `@typescript/native`). That keeps lint on TS6 and was the pre-PR state. Bumping the deferred `typescript` package itself to 7 required leaving `typescript-eslint`.

## Next check dates

| When | What to check |
|------|----------------|
| **2026-09-09** | TypeScript **7.1 Beta** (planned) — new tooling API surface |
| **2026-10-20** | TypeScript **7.1 RC** |
| **2026-11-10** | TypeScript **7.1 Stable** |
| After each | `typescript-eslint` peer range + `#10940` / draft native backend `#12803` |

If/when `typescript-eslint` supports TS ≥7.1, evaluate restoring ESLint typed rules or keeping oxlint (already TS7-native via tsgolint).
