# 12. Theming and Tailwind

Every visual aspect of GoUI's built-in form controls — colors, radius,
spacing — is driven by a small set of CSS custom properties ("design
tokens") defined once, in one file. Rebranding GoUI is a CSS-only exercise;
you never edit Go code or component templates to change how things look.

Module path used throughout this document: `github.com/zatrano/goui`.

## 1. The token set

All tokens live in `:root` at the top of `forms/style.css`:

```css
:root {
  --color-goui-primary: oklch(55% 0.18 250);
  --color-goui-border: oklch(85% 0.01 250);
  --color-goui-error: oklch(55% 0.22 25);
  --color-goui-success: oklch(52% 0.14 145);
  --color-goui-warning: oklch(70% 0.15 75);
  --color-goui-info: oklch(55% 0.12 250);
  --color-goui-text: oklch(20% 0.01 250);
  --color-goui-surface: oklch(99% 0.005 250);
  --radius-goui: 0.375rem;
  --spacing-goui-field: 0.75rem;
}
```

| Token | Used for | Consumers |
|---|---|---|
| `--color-goui-primary` | Buttons, focus rings, accent inputs, selected/active states | `.bg-goui-primary`, `.accent-goui-primary`, calendar day selection, chips, swatches |
| `--color-goui-border` | Default borders on inputs, fieldsets, panels | `.border-goui-border`, most `goui-*` component borders |
| `--color-goui-error` | Invalid field state, error toasts | `.text-goui-error`, `.border-goui-error`, `.goui-toast-error`, password-strength "weak" |
| `--color-goui-success` | Success toasts | `.goui-toast-success` |
| `--color-goui-warning` | Warning toasts | `.goui-toast-warning` |
| `--color-goui-info` | Info toasts (also the fallback/default kind) | `.goui-toast-info` |
| `--color-goui-text` | Body/label text color, and the base for most `color-mix()` derived shades (muted text, hover tints, shadows) | `.text-goui-text`, labels, helper text, shadows |
| `--color-goui-surface` | Backgrounds of inputs, panels, dropdowns, toasts | `.goui-input`, `.goui-searchable-panel`, `.goui-toast` |
| `--radius-goui` | Corner radius across almost every control | `.rounded-goui`, buttons, chips, calendar cells |
| `--spacing-goui-field` | Horizontal/vertical field padding and gaps | `.px-goui-field`, `.py-goui-field`, `.gap-goui-field` |

Two things worth noting about the palette's shape:

- Colors are defined in **`oklch()`** (lightness%, chroma, hue), which is
  why so much of the rest of the stylesheet layers `color-mix(in oklch,
  var(--color-goui-...) X%, white)` to derive hover/muted/tint shades instead
  of hardcoding separate variables for every shade — perceptually uniform
  lightening/darkening comes almost for free with `oklch`.
- Only the tokens above are load-bearing for GoUI's own components. Utility
  classes like `.w-full`, `.flex`, `.text-sm`, etc. (also in `forms/style.css`)
  are plain fixed CSS, not tokenized — they exist so the bundled examples
  don't need a full Tailwind build to look reasonable out of the box (see
  §3 for wiring real Tailwind on top).

## 2. Overriding tokens for your own brand

Because every component reads tokens through `var(--color-goui-...)`,
rebranding is just re-declaring the same custom property names, at a point
in the CSS cascade **after** `forms/style.css` is loaded (a later same-
specificity `:root` rule wins, or scope it to `body`/a wrapper class if you
need higher specificity).

Worked example — a fictional "RenewOS" brand with a green primary and a
warmer neutral palette:

```css
/* renewos-theme.css — load this AFTER forms/style.css */
:root {
  --color-goui-primary: oklch(58% 0.16 155);   /* RenewOS green */
  --color-goui-border:  oklch(88% 0.01 90);    /* warm light gray */
  --color-goui-error:   oklch(56% 0.21 22);
  --color-goui-success: oklch(60% 0.15 150);
  --color-goui-warning: oklch(74% 0.14 70);
  --color-goui-info:    oklch(60% 0.10 220);
  --color-goui-text:    oklch(22% 0.015 90);
  --color-goui-surface: oklch(98% 0.006 90);
  --radius-goui: 0.5rem;               /* slightly rounder than the default 0.375rem */
  --spacing-goui-field: 0.875rem;      /* a touch more breathing room */
}
```

```html
<link rel="stylesheet" href="/forms/style.css">
<link rel="stylesheet" href="/assets/renewos-theme.css">
```

That's the entire rebrand: every button, input border, toast, calendar
selection, chip, and focus ring across `forms/*` picks up the RenewOS
palette immediately, with zero changes to any
component's Go code or rendered markup. You can scope this further — e.g.
`.theme-renewos { --color-goui-primary: ...; }` on a wrapper element — if a
single page needs to host more than one theme at once (a white-labeled
multi-tenant admin panel, for instance).

Only override the tokens you actually want to change; anything you don't
redeclare keeps the default value from `forms/style.css`.

## 3. Tailwind integration

`forms/style.css` and its tokens work perfectly well with **no Tailwind at
all** — that's what the plain `<link rel="stylesheet">` approach above does.
Tailwind is entirely optional and is only useful if you want to author
*your own* application markup (outside the GoUI form controls) using
Tailwind utility classes that share the same brand tokens.

GoUI's core has **no npm dependency and no `package.json`** — Tailwind is
brought in, if at all, only inside an example/app directory, using
Tailwind v4's standalone CLI via `npx` (no local npm project required):

```css
/* examples/contact-form/input.css */
@import "../../forms/style.css";

@theme {
  --color-goui-primary: oklch(55% 0.18 250);
  --color-goui-border: oklch(85% 0.01 250);
  --color-goui-error: oklch(55% 0.22 25);
  --color-goui-text: oklch(20% 0.01 250);
  --radius-goui: 0.375rem;
  --spacing-goui-field: 0.75rem;
}
```

```bat
:: examples/contact-form/build-css.bat
@echo off
REM Optional: build Tailwind utilities from input.css
REM Requires: npx (or standalone tailwindcss binary)
npx --yes @tailwindcss/cli@4 -i ./input.css -o ./output.css
echo Built output.css — point index.html at it if you want Tailwind-generated utilities.
```

Run it (Windows):

```powershell
cd examples\contact-form
.\build-css.bat
```

which is equivalent to, and can be run directly, cross-platform:

```bash
npx --yes @tailwindcss/cli@4 -i ./input.css -o ./output.css
```

What's happening here:

1. `@import "../../forms/style.css";` pulls in GoUI's base rules and default
   tokens first.
2. The `@theme { ... }` block is Tailwind v4's mechanism for registering
   design tokens as **first-class Tailwind theme values** — declaring
   `--color-goui-primary` inside `@theme` means Tailwind will also generate
   matching utilities for it (e.g. `bg-[color:var(--color-goui-primary)]`-
   style access, or, if you mirror the token under Tailwind's own naming
   convention, plain utilities like `bg-goui-primary`). This is how you keep
   one source of truth for brand colors across both GoUI's own controls and
   any hand-written Tailwind markup around them.
3. `@tailwindcss/cli@4` is invoked through `npx --yes`, which downloads and
   runs the package on demand — **no `npm install`, no `node_modules`, no
   `package.json` anywhere in the repository.** If you'd rather not depend
   on `npx` reaching the network at build time, download the
   [standalone Tailwind CLI binary](https://tailwindcss.com) for your
   platform once and invoke it directly in place of `npx --yes
   @tailwindcss/cli@4`; the `-i`/`-o` arguments are identical either way.
4. The generated `output.css` is a static file — commit it or regenerate it
   as part of your own build/deploy step; GoUI itself never invokes Tailwind
   or Node at runtime, in tests, or anywhere in `go build`.

### 3.1 Applying your own brand through the Tailwind path

To rebrand *and* keep using Tailwind, change the values inside your own
`input.css`'s `@theme` block (mirroring the plain-CSS override from §2),
then rebuild:

```css
/* input.css, RenewOS variant */
@import "../../forms/style.css";

@theme {
  --color-goui-primary: oklch(58% 0.16 155);
  --color-goui-border:  oklch(88% 0.01 90);
  --color-goui-error:   oklch(56% 0.21 22);
  --color-goui-text:    oklch(22% 0.015 90);
  --radius-goui: 0.5rem;
  --spacing-goui-field: 0.875rem;
}
```

```bash
npx --yes @tailwindcss/cli@4 -i ./input.css -o ./output.css
```

Serve `output.css` instead of (or after) plain `forms/style.css`, and both
GoUI's built-in controls and any Tailwind utility classes in your own
templates share the RenewOS palette.

## 4. Recommendations

- **Start without Tailwind.** Link `forms/style.css` plus your own token
  overrides. Add Tailwind only once you're writing enough custom
  application markup around GoUI's components to want utility classes.
- **Keep one token override file per brand/tenant**, loaded after
  `forms/style.css`, rather than editing `forms/style.css` itself — this
  keeps you trivially upgradable when the base stylesheet changes.
- **Don't fork component Go code to change styling.** Every `forms/*`
  control that needs conditional styling (invalid state, disabled state,
  selection) already expresses it through classes/tokens
  (`FieldValidation.ApplyErrorState`, `border-goui-border`/`border-goui-error`,
  etc.) — if you find yourself wanting to edit a `.go` file just to change a
  color, look for the token first.
