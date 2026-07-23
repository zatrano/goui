# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

## [1.0.0] — 2026-07-23

### Added

- **Page modes** (`ModeLive` / `ModeSEO` / `ModeStatic`): register with
  `Registry.RegisterPage`, serve full documents via `page.NewRenderer` and
  adapter `Routes` / `Page(...)`.
- `core.Head` / `HeadProvider` for title, description, Open Graph, robots.
- `core.ContextWithRequest` / `RequestFromContext` for SEO Mount access to
  `*http.Request`.
- Client hydrate: adopt `[data-goui-ssr]` on first WS render (no flash).
- Docs: [docs/en/17-page-modes.md](docs/en/17-page-modes.md) (TR mirror).
- Example: [`examples/seo-pages`](examples/seo-pages).
- Updated project banner to show WebSocket and SEO HTML delivery paths.

## [0.2.0] — 2026-07-21

Blade-inspired file-based template engine (`template` package).

### Added

#### Template engine
- Lexer / parser / AST for `@directive` syntax (keeps Go `{{ }}` expressions intact)
- Codegen onto native `html/template` with auto-escaping (`raw`, `dict`, `list`, `default`)
- `Registry`: load `.goui.html`, `@extends` / `@section` / `@yield`, `@include` / `@includeIf`
- Components: `@component` / `@slot` / `@props`, `component` FuncMap helper, `Dot` / slots
- Hot reload via `fsnotify` (`WatchForChanges`, debounce, keep last-good tree on error)
- `ViewComponent` + `template.Wrap` / `RenderComponent` integration with core
- Opt-in `StrictProps` (typo suggestions + unused-prop warnings)
- Docs: [docs/en/15-template-engine.md](docs/en/15-template-engine.md), migration [docs/en/16-migrating-to-template-engine.md](docs/en/16-migrating-to-template-engine.md) (TR mirrors)
- Example: `examples/counter-view`
- E2E + benchmarks under `template/`

### Dependencies

- `github.com/fsnotify/fsnotify` (hot reload)

## [0.1.0] — 2026-07-16

Initial public-ready snapshot.

### Added

#### Core
- `Component` lifecycle and `BaseComponent` (dirty tracking)
- `Registry` and HTML `RenderTemplate` cache
- Counter example

#### i18n & WebSocket
- `i18n.Translator`, JSON locales (`tr` base, `en`)
- `Session`, `Hub`, WebSocket sessions on `/goui/ws`
- Reconnect + grace period (default 60s)
- Frame types: `event`, `render`, `error`, `session`

#### Diff
- HTML parse / serialize, patch ops, keyed list diff (`data-key`)

#### Client runtime
- `client/goui.js`: patches, event delegation, reconnect

#### Forms & validation
- Native and rich form controls in the `forms` package
- `validation` rules with server-side checks and preserved component state
- Advanced selects, date/time, currency/rating, uploads, editors, identity pickers, visual pickers

#### Toast / push
- `Toast` / `ToastT`, `Hub.Broadcast`, client toast stack

#### Prefetch
- `prefetch` / `activate` frames, silent Mount, LRU=5
- Client `data-goui-prefetch` / `data-goui-activate`

### Changed

- Public module path is now `github.com/zatrano/goui`; all form controls
  live in the single `forms` package.
- Public interfaces use `client/goui.js` / `GoUIClient`,
  `/goui/ws` + `/goui/upload` + `/goui/files`, `data-goui-*`, `g-*` event
  bindings, and `.goui-uploads`.

### Known limitations

- Some advanced form controls are not implemented yet
- Upload: `LocalStore` only (S3/MinIO via `upload.Storage` TBD)
