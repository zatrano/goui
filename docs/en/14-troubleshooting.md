# 14. Troubleshooting

A checklist-style reference for the failure modes that come up most often
when building on GoUI. Each section is meant to be skimmable during an
incident — jump to the symptom you're seeing.

Module path used throughout this document: `github.com/zatrano/goui`.

---

## 1. WebSocket not connecting

Symptoms: the browser never receives a `"session"` frame, `onConnect` never
fires, `onError` fires immediately, or the socket closes right after
opening.

Work through these in order:

- [ ] **Is `/goui/ws` actually reachable?** Your adapter mounts
  `GET /goui/ws` (`ws.Path`) and rejects non-WebSocket-upgrade requests
  (Fiber returns `fiber.ErrUpgradeRequired`; stdlib/Gin/Echo use their
  own upgrade checks — see `adapters/`). Hitting the
  URL with a plain browser navigation/`fetch` will correctly fail — that's
  not a bug. Confirm the *client* is actually constructing a `ws://`/`wss://`
  URL, not `http://`/`https://`.
- [ ] **Is the URL same-origin/relative, or did you hardcode a scheme/host
  that doesn't match how the page is served?** `GoUIClient._buildUrl` only
  auto-derives `ws:`/`wss:` when the configured `wsUrl` does **not** already
  start with `ws` — if you pass a fully-qualified `http://` URL by mistake,
  it will not be corrected for you.
- [ ] **Behind a reverse proxy?** Nginx (and most proxies) do **not**
  forward the `Upgrade`/`Connection: Upgrade` headers unless you explicitly
  configure `proxy_http_version 1.1;`, `proxy_set_header Upgrade
  $http_upgrade;`, and `proxy_set_header Connection $connection_upgrade;`.
  Missing this is the single most common deployment-time cause of "it works
  on localhost but not in production." See
  [13-project-integration.md](13-project-integration.md) §6.3 for a
  complete, working Nginx config.
- [ ] **Was `?component=` omitted on a fresh connection with no
  `?session=`?** The server responds with an error frame
  (`ws.ErrComponentRequired`, `"component query parameter is required"`)
  and closes the connection if neither a known `session` nor a
  `component` query parameter is supplied. Confirm `GoUIClient` was
  constructed with a non-empty `componentName`.
- [ ] **Is the component name actually registered?** `registry.Create`
  returns `core.ErrComponentNotRegistered` for an unknown name, which the
  server turns into an error frame and then closes the connection without
  ever sending a `"session"` frame. Double-check the exact string passed to
  `registry.Register(...)` matches what the client requests — this is
  especially easy to get wrong with the tenant-qualified names described in
  [13-project-integration.md](13-project-integration.md) §2.
- [ ] **Check the browser's Network/WS inspector for the actual close code
  and any HTTP response before the upgrade.** A `101 Switching Protocols`
  that never arrives (stuck at `pending`, or a `4xx`/`5xx` instead) points
  at the proxy/routing layer; a socket that opens and then immediately
  closes points at the error-frame path above — read the payload of the
  `"error"` frame before it closes, if any arrived.
- [ ] **Server-side logs.** `Session.readLoop` logs prefetch failures
  (`log.Printf("[goui] prefetch %q failed: %v", ...)`) but most connection-
  level failures are only visible as the error frame sent to the client —
  there is no separate server-side WS connection log by default. Add your
  own logging around `registry.Create`/`hub.Get` calls if you need
  server-side visibility into repeated failures.

---

## 2. Stale session (`sessionStorage` key `goui.sessionId`)

The client persists its session ID in `sessionStorage` under the key
`goui.sessionId` (`SESSION_KEY` in `client/goui.js`) so that a page
reload — not just a network blip — can reattach to the *same* server-side
`Session` (and therefore the same mounted component state) instead of
starting fresh.

**Symptom:** after a server restart (see §6) or a `Hub` recreation, the
browser still has an old `goui.sessionId` in `sessionStorage`, tries to
reconnect with `?session=<old-id>`, and the server — which has no memory of
that ID anymore — responds `"session not found"` and closes the connection
without ever registering a new session.

The client already has a self-healing path for exactly this case — verify
it's actually reachable in your setup:

```js
// client/goui.js — _handleFrame, case 'error'
if (message === 'session not found' || message.includes('session not found')) {
  this.sessionId = '';
  sessionStorage.removeItem(SESSION_KEY);
  this.componentRoots.clear();
  if (this.ws) this.ws.close();
  this.onError(message + ' — reconnecting fresh');
  return;
}
```

Checklist if this *isn't* recovering automatically:

- [ ] **Is `onclose` actually wired to `_scheduleReconnect`?** If you
  overrode or bypassed `GoUIClient`'s built-in reconnect logic (custom
  transport wrapper, manual `WebSocket` usage, etc.), the automatic
  "clear stale ID and reconnect fresh" path above never runs — you'll need
  to reproduce it.
- [ ] **Is something else holding a copy of the old session ID** (a
  server-rendered page that embedded it at load time, a cookie, custom
  storage) and re-injecting it after the client already cleared
  `sessionStorage`? Confirm the *only* source of truth for `?session=` on
  reconnect is `sessionStorage.getItem('goui.sessionId')` read fresh at
  `_buildUrl()` time.
- [ ] **Manually clearing state for a "log out" / "switch tenant" flow?**
  Call `sessionStorage.removeItem('goui.sessionId')` (or
  `sessionStorage.clear()`) yourself before reconnecting with a different
  component/tenant name — otherwise the browser will try to reattach to a
  session that (correctly) has no relationship to the new context you want.
- [ ] **Multiple tabs sharing a session ID unexpectedly?** `sessionStorage`
  is per-tab by spec, but if you copied the ID into `localStorage` or a
  cookie somewhere in your own code, two tabs can race to reattach to the
  same server `Session`, and `ws.ErrSessionAlreadyActive` will be returned
  to whichever one loses the race (`Session.Reattach` refuses to reattach
  if a connection is already active). Keep session IDs scoped to
  `sessionStorage` unless you specifically intend to share one session
  across tabs (which GoUI does not support out of the box).

---

## 3. Patch path looks "off by one" / patches land on the wrong element

Symptom: a `set_attr`/`update_text`/`replace` patch appears to target a
sibling, or nothing visibly happens, or an error appears about a missing
target.

- [ ] **Remember paths are relative to the component root, resolved via
  "meaningful children," not raw DOM `childNodes`.** Both the server
  (`diff.ParseHTML`/`convertHTMLNode`) and the client
  (`meaningfulChildren()` in `client/goui.js`) drop whitespace-only text
  nodes before indexing children. If you're comparing indices against raw
  browser DevTools `childNodes` output (which *does* include whitespace
  text nodes from indentation in your HTML string), the indices will not
  line up — always reason about indices in terms of *elements + non-blank
  text*, matching what `meaningfulChildren` computes.
- [ ] **Did `Render()` return more than one root element?** If so, GoUI
  wraps your output in a synthetic `<div data-goui-component="...">` (see
  [10-diffing-internals.md](10-diffing-internals.md) §3), which shifts
  every path down one level of nesting relative to what you might expect
  from reading your own `Render()` output in isolation. Fix by making
  `Render()` return exactly one root element — this both removes the extra
  nesting and matches the element the client looks up by
  `[data-goui-component]`.
- [ ] **Is the previous render tree actually the one being diffed against?**
  `Session.renderTrees[componentID]` is only set by `sendRender`/
  `sendFullRender` after a successful `Render()` + parse. If a prior
  `Render()` call errored partway through your own request handling (and
  you swallowed the error), the stored tree can be older than what's
  currently in the live DOM, producing patches that look correct against
  the *stored* tree but wrong against what the user is actually looking at.
  Make sure every `HandleEvent` path either returns an error (surfaced to
  the client as an error frame — see `Session.handleEventFrame`) or leaves
  the component in a state whose next `Render()` output matches reality.
- [ ] **Did you hand-edit the DOM outside of GoUI** (browser extension,
  another script, manual DevTools edit) within a component's subtree that
  *isn't* marked `data-goui-ignore`? GoUI's diffing assumes the live DOM
  matches the last tree it rendered; any out-of-band mutation inside a
  reconciled subtree can desync indices for subsequent patches. Wrap any
  intentionally client-owned region in `data-goui-ignore` (§5) instead of
  letting GoUI reconcile over it.

---

## 4. Checkbox/radio state doesn't visually match after a patch

Symptom: the server clearly sent `set_attr checked="checked"` (or removed
it), but the checkbox/radio's visual state in the browser doesn't change to
match — or changes, then flips back on the next unrelated interaction.

This is a well-known DOM quirk, not a GoUI bug, but it's worth knowing
exactly how GoUI handles it so you can recognize when the handling itself is
being bypassed: for `checked`, `selected`, `disabled`, and `readonly`,
**HTML attributes and DOM properties are two different things** once a
checkbox has been interacted with — setting the `checked` *attribute* via
`setAttribute` does not reliably update the *live* `.checked` *property* the
browser actually renders from. `client/goui.js`'s `applyPatch` handles this
explicitly for both `set_attr` and `remove_attr`:

```js
case 'set_attr': {
  const target = resolvePath(rootEl, path);
  if (target && target.nodeType === Node.ELEMENT_NODE) {
    const name = patch.attr;
    const value = patch.value ?? '';
    target.setAttribute(name, value);
    // Boolean DOM properties must stay in sync for form controls after patch.
    if (name === 'checked' || name === 'selected' || name === 'disabled' || name === 'readOnly' || name === 'readonly') {
      const prop = name === 'readonly' ? 'readOnly' : name;
      target[prop] = true;
    }
    if (name === 'value' && 'value' in target) {
      target.value = value;
    }
  }
  break;
}
```

Checklist if you're still seeing a mismatch:

- [ ] **Are you patching a subtree marked `data-goui-ignore`?** `applyPatch`
  bails out before reaching the code above if the target (or its parent,
  for insert/remove) is inside an ignored region (§5) — intentionally, so
  client-owned widgets aren't touched. If your checkbox is inside such a
  region for an unrelated reason, its `checked` state will never be synced
  by GoUI at all; it's entirely up to whatever owns that region.
- [ ] **Did the patch actually reach the browser?** Confirm via the network
  inspector (or a temporary `onError`/console log) that a `set_attr`/
  `remove_attr` patch for `checked`/`value` was actually sent for this
  event — if `MarkDirty()` was never called after mutating server-side
  state, no render (and therefore no patch) is produced at all; the
  component's *next* unrelated render will eventually reflect the change,
  which can look like "it fixed itself on the next click."
  `forms.ChoiceInput.HandleEvent` calls `MarkDirty()` unconditionally on a
  recognized change event — if you built a custom checkbox control, make
  sure yours does too.
- [ ] **Is a `value`/`checked` value being set on an element type where that
  property doesn't apply** (e.g. setting `checked` on a plain `<div>`
  instead of an `<input>`)? The property-sync branch above only assigns
  `target[prop]` — it doesn't verify the element type supports that
  property; assigning `checked` to a non-form element is silently a no-op
  from the DOM's perspective, which can look identical to "the patch didn't
  apply."

---

## 5. Rich text / code editor cursor jumps, or edits get clobbered

Symptom: typing in a Quill (`forms.RichTextEditor`) or CodeMirror
(`forms.CodeEditor`) instance causes the cursor to jump to the start/end of
the field, selections get lost, or the whole editor visibly re-mounts while
typing.

This happens when GoUI's diff/patch reconciliation is allowed to touch DOM
that a third-party editor library owns and mutates on its own. The fix is
always some combination of the same three mechanisms — check that all three
are actually in place for your control:

- [ ] **`data-goui-ignore` on the editor's mount element.** Both
  `RichTextEditor.Render()` and `CodeEditor.Render()` set
  `data-goui-ignore="1"` on their outer wrapper. `client/goui.js`'s
  `isGoUIIgnored`/`applyPatch` skip any patch whose resolved target (or, for
  insert/remove, whose resolved *parent*) is inside a `[data-goui-ignore]`
  ancestor — meaning GoUI's reconciler will never touch anything inside that
  element after the first render, no matter what changes server-side. If
  you're writing a custom third-party-widget wrapper, copy this attribute
  onto its outer element.
- [ ] **A stable, "empty-on-render" sync channel, not the live value.**
  Neither `RichTextEditor` nor `CodeEditor` renders the current `Value`
  into the ignored subtree on every render — the sync `<textarea
  class="goui-editor-sync">` is deliberately always rendered empty, with the
  *initial* value only available via `data-initial` (read once, at mount
  time, by the JS bridge — see `client/modules/richtext.js`'s
  `mountQuill`). If a custom control instead re-renders the widget's
  *current* text into markup on every server round trip, you will fight the
  diff patcher even with `data-goui-ignore` in place, because a `data-
  goui-ignore`d element that keeps changing shape upstream of it can still
  cause the *parent* to re-key/replace it.
- [ ] **Return `core.ErrSkipRender` from `HandleEvent` for pure DOM-sync
  events, if a render isn't otherwise needed.** `Session.handleEventFrame`
  checks for this sentinel explicitly:

  ```go
  if err := component.HandleEvent(ctx, frame.Event, payload); err != nil {
      if errors.Is(err, core.ErrSkipRender) {
          return // acknowledge the event, but do not render/patch anything
      }
      // ...
  }
  ```

  Note that `RichTextEditor`/`CodeEditor`'s own `HandleEvent` implementations
  currently just update `Value` and skip `MarkDirty()` (relying on the
  parent form to decide whether/when to re-render) rather than returning
  `ErrSkipRender` directly — either approach avoids an unnecessary
  patch cycle for a value that lives entirely in ignored, client-owned DOM.
  Use `ErrSkipRender` explicitly in your own event handlers whenever an
  event exists purely to keep server-side state in sync with a client-owned
  widget and a re-render would provide no visible benefit (and risks
  disturbing cursor/selection state if the ignored region's *ancestor*
  happens to be replaced for an unrelated reason).
- [ ] **Debounce the sync event.** Both editors set `g-debounce` (default
  350ms) on their sync `<textarea>`'s `g-input` binding — sending a WS event
  (and therefore a `HandleEvent` call) on every single keystroke is wasteful
  even when it doesn't trigger a visible render; keep a debounce in place
  for any similar free-typing sync channel.

---

## 6. "I changed the component code but nothing changed" (server restart)

Symptom: you edited a component's `Render`/`HandleEvent`/`Mount` (or any
other Go source), reloaded the browser tab, and the app still behaves like
the old code.

This one has a single cause worth internalizing, since it's the most common
false alarm when developing against GoUI: **GoUI components are compiled Go
code, not templates or interpreted scripts.** Unlike the client-side JS in
`client/*.js` (which the browser re-fetches on a normal page reload), a
change to any `.go` file — a component, a validation rule, a registered
factory, the registry wiring itself — requires the Go process to be
**rebuilt and restarted** before it takes effect. There is no hot-reload
inside GoUI itself.

Checklist:

- [ ] **Did you actually rebuild?** `go run ./cmd/myapp` rebuilds on every
  invocation; a long-running `go build && ./bin/myapp` workflow does not —
  if you're iterating, either re-run `go build` each time or use a file
  watcher (`air`, `wgo`, `entr`, `reflex`, etc.) that rebuilds *and*
  restarts the binary on save.
- [ ] **Did the restarted process actually bind the port** you're hitting
  (no `"address already in use"` from a previous instance still running)?
  A failed restart that silently leaves the old process serving traffic
  looks identical to "my change had no effect."
- [ ] **Did existing browser tabs reattach to a session hosted by the *old*
  process, then get a `"session not found"` from the *new* one?** After a
  restart, the in-memory `*ws.Hub` is empty — every previously-connected
  browser tab's `goui.sessionId` now refers to a session that doesn't exist
  anymore. Per §2, the client already recovers automatically (clears the
  stale ID, reconnects fresh, remounts). If you don't see a fresh mount
  happen, hard-reload the tab (bypassing any cached JS) rather than
  concluding the server-side change didn't take effect.
- [ ] **Are you editing a vendored/copied checkout of the `goui` module
  rather than the one actually imported by your `go.mod`?** If your app
  depends on `github.com/zatrano/goui` via a version pin (not a local
  `replace` directive pointing at your working copy), editing files inside
  a separate local clone has no effect on the binary until you either add a
  `replace github.com/zatrano/goui => ../path/to/goui` in `go.mod` for local
  development, or bump/re-vendor the dependency.
- [ ] **Did you change styling only (`forms/style.css`, a theme override) and
  expect a Go rebuild?** CSS and the client JS modules under `client/` are
  static files served as-is — those *do* take effect on a plain browser
  reload (or a hard reload if the browser cached them aggressively). Don't
  restart the Go process chasing a CSS change; conversely, don't expect a
  browser reload alone to pick up a Go-side change.
