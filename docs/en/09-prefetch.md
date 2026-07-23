# 09. Prefetch

Prefetch lets the browser ask the server to run a component's `Mount(ctx)`
*before* the user actually navigates to it, so that when they do, the first
render is instant instead of waiting on a round trip plus whatever work
`Mount` does (database reads, cache warm-up, etc.).

Module path used throughout this document: `github.com/zatrano/goui`.

## 1. The two attributes

Prefetch is entirely attribute-driven; you don't call any JS API directly in
most apps.

```html
<a href="/contact" data-goui-prefetch="contact" data-goui-activate="contact">
  Go to contact form
</a>
```

- **`data-goui-prefetch="<registry-name>"`** — marks an element as a prefetch
  trigger for the component registered under `<registry-name>`. The client
  module `client/modules/prefetch.js` prefetches it when:
  - the pointer hovers the element for **~100ms** (`HOVER_MS`), or
  - the element scrolls into the viewport, via `IntersectionObserver` with
    an `80px` root margin — i.e. it starts loading slightly before it's
    actually visible.

  Each name is only ever requested once per client session
  (`prefetch.js` tracks a `requested` `Set`), so re-hovering or re-scrolling
  past the same link is a no-op on the client side.

- **`data-goui-activate="<registry-name>"`** — on click, prevents the default
  navigation, immediately requests a prefetch (if not already requested) and
  sends an **activate** frame for that name. This is the attribute that
  actually promotes the prefetched component into a rendered, visible one.

You can use `data-goui-prefetch` alone (just warm it up, activate through
some other mechanism/event later), or combine both on the same element as
shown above — the common "warm on hover, swap in on click" pattern used by
`examples/contact-form`'s `Landing` component:

```go
func (l *Landing) Render() (string, error) {
	return `<div class="landing">
  <p><a href="#" data-goui-prefetch="contact" data-goui-activate="contact">Go to the contact form</a></p>
</div>`, nil
}
```

## 2. Wire protocol

Two new frame types, defined in `ws/frame.go`:

```go
const (
	// ...
	FrameTypePrefetch = "prefetch"
	FrameTypeActivate = "activate"
)
```

The client (`client/goui.js`) sends them as plain frames carrying only a
component *type name* (not yet an instance ID, since the component doesn't
exist server-side until `Prefetch`/`Activate` runs):

```js
sendPrefetch(componentName) {
  this.ws.send(JSON.stringify({ type: 'prefetch', component: componentName }));
}

sendActivate(componentName) {
  this.ws.send(JSON.stringify({ type: 'activate', component: componentName }));
}
```

On the server, `Session.readLoop` (`ws/session.go`) dispatches these to
`Session.Prefetch` and `Session.Activate` respectively.

## 3. What happens on the server

### 3.1 `Session.Prefetch(name)`

```go
func (s *Session) Prefetch(name string) error {
	// no-op if name is empty, or already prefetched
	// registry.Create(name) → prepareComponent (SetTranslator, SetPusher, Mount(ctx))
	// insert into s.prefetched[name] and append to s.prefetchOrder
	// evict oldest if len(s.prefetchOrder) >= MaxPrefetch
}
```

Key properties:

- It **creates and `Mount`s** a real component instance via the session's
  `*core.Registry`, with the translator and toast pusher already wired up
  exactly as a normally-activated component would have.
- It **does not render** and **does not send any frame** to the client.
  Prefetching a component produces zero WebSocket traffic beyond the
  prefetch request itself — confirmed by the test suite
  (`TestSession_Prefetch_MountsWithoutRender` asserts no outbound frame at
  all after a `Prefetch` call).
- Duplicate prefetches for the same name are a no-op: if `name` is already
  in `s.prefetched`, `Prefetch` returns immediately without creating a
  second instance or calling `Mount` again.
- The instance lives in `s.prefetched[name]`, a map keyed by **registry
  name**, not by a generated component ID — there is intentionally no
  server-assigned ID yet, because the component isn't "live" until activated.

### 3.2 `Session.Activate(name)`

```go
func (s *Session) Activate(name string) (string, error) {
	// if name is in s.prefetched: reuse that instance (delete from prefetched map)
	// else: registry.Create(name) fresh, then Mount it
	// assign a fresh component ID, store in s.components[id]
	// sendFullRender(id, component)
	return id, nil
}
```

- If the name was previously prefetched, `Activate` **reuses the exact same
  instance** — no second `Mount` call, and any state accumulated during
  `Mount` (or even mutated later, though nothing should mutate a prefetched
  component before activation) is preserved. This is verified by
  `TestSession_Prefetch_ActivateUsesExisting`.
- If the name was *not* prefetched (e.g. the user clicked before the 100ms
  hover timer or the intersection observer fired, or `data-goui-activate` was
  used without `data-goui-prefetch`), `Activate` transparently falls back to
  creating and mounting a fresh instance on the spot — from the caller's
  point of view, prefetching is purely an optimization; correctness doesn't
  depend on it.
- Either way, activation is the point at which the component gets a real
  component ID and its **first full render** is sent
  (`Op: diff.OpReplace, Path: []`), exactly like any other newly-mounted
  component's initial render.

## 4. `MaxPrefetch` and LRU eviction

```go
// ws/frame.go
// MaxPrefetch is the per-session cap for silently mounted (not yet visible) components.
const MaxPrefetch = 5
```

Each session may hold at most 5 prefetched-but-not-yet-activated components
at once. `s.prefetchOrder` tracks insertion order (oldest first); when a 6th
distinct prefetch comes in, the oldest entry is evicted:

```go
for len(s.prefetchOrder) >= MaxPrefetch {
	oldest := s.prefetchOrder[0]
	s.prefetchOrder = s.prefetchOrder[1:]
	if old, ok := s.prefetched[oldest]; ok {
		delete(s.prefetched, oldest)
		evicted = append(evicted, old)
	}
}
```

Evicted components are properly `Unmount(ctx)`ed (outside the session lock,
after eviction bookkeeping) — this is not a leak, it's a real teardown, so
`Mount`/`Unmount` should be written as a matched pair regardless of whether a
component is ever activated. This cap exists to bound how much speculative
work a single browser tab can force the server to do (e.g. a user hovering
over ten navigation links in a row shouldn't leave ten live DB connections
warm on the server indefinitely).

Prefetched components that are *never* activated are also cleaned up when
the session itself expires past the WebSocket grace period — see
[13-project-integration.md](13-project-integration.md) §7 and
`TestSession_Prefetch_CleanedOnGracePeriodExpiry`.

## 5. When to use prefetch

Prefetch is a good fit when:

- `Mount` does real, possibly-slow work (a DB query, an external API call, a
  cache lookup) that you'd like to have already finished by the time the
  user clicks.
- The destination is *likely* to be visited — primary navigation, "next
  step" links in a wizard, tabs, a details view opened from a hover card.
- `Mount` is **idempotent and side-effect-free** — it's fine to call it and
  then simply throw the result away (via eviction or a session that
  disconnects) without anything bad happening.

## 6. When *not* to use prefetch

- **`Mount` has side effects.** If mounting a component sends an email,
  inserts an audit-log row, increments a counter, acquires an exclusive
  resource, or does anything else that shouldn't happen "just because the
  mouse hovered over a link," do not prefetch it. Remember: a prefetched
  component's `Mount` really does run, in full, on the server, even though
  the user may never click through.
- **Rarely-visited destinations.** Prefetching a link nobody clicks wastes a
  `Mount`/`Unmount` cycle and occupies one of the 5 prefetch slots (evicting
  something that might have actually been useful).
- **Already-cheap `Mount`.** If mounting is just zeroing a struct field,
  prefetching adds a network round trip and bookkeeping for no measurable
  benefit — just activate directly.
- **Highly dynamic component names.** Prefetch identifies components purely
  by registry name. If the actual component you want to show depends on
  request-time parameters that aren't captured in that name (a row ID, a
  search query), prefetching the generic name may warm the wrong thing, or
  nothing useful at all. Prefetch is best suited to a fixed set of named
  destinations (tabs, wizard steps, common pages), not per-record detail
  views (unless you register one name per record, which usually isn't
  practical).

## 7. No pre-render — why that matters

It's worth repeating because it's easy to assume otherwise: prefetch **never
renders** and **never sends HTML** to the browser. All it buys you is that,
at activation time, `Mount` has already happened. The activation call still
does:

1. Assign a component ID.
2. Call `Render()`.
3. Wrap/decorate the HTML with `data-goui-component="<id>"`.
4. Parse it into a `diff.Node` tree.
5. Send it as a single `OpReplace` patch at the root path.

So prefetch shaves the `Mount` latency off the critical path, but the
render/serialize/parse/send steps still happen synchronously at activation
time, same as any other first render. If your bottleneck is a slow `Render()`
rather than a slow `Mount()`, prefetch won't help — optimize `Render()`
itself, or reduce what it needs to compute.
