# 16 — Migrating to the template engine

This guide moves components from inline `core.RenderTemplate` strings to
`.goui.html` files and `core.ViewComponent`.

**Backward compatibility:** `core.RenderTemplate` is **not** removed. Existing
components keep working. Migration is optional.

## When it is worth migrating

| Stay on `RenderTemplate` | Move to `.goui.html` |
|--------------------------|----------------------|
| Tiny one-liner markup | Multi-section pages / layouts |
| No reuse across components | Shared partials / components / slots |
| Prototype / throwaway demo | Production UI you will edit often |

## Step by step

### 1. Before

```go
func (c *Counter) Render() (string, error) {
    return core.RenderTemplate(`<div class="counter">
<span>{{.Count}}</span>
<button g-click="increment">+</button>
</div>`, c)
}
```

### 2. Extract the view

Create `views/counter.goui.html`:

```html
<div class="counter">
  <span>{{ .Count }}</span>
  <button type="button" g-click="increment">+</button>
</div>
```

### 3. Implement `View` (+ stub `Render`)

```go
func (c *Counter) View() string { return "counter" }

func (c *Counter) Render() (string, error) {
    return "", gouitemplate.ErrViewRenderDirect
}
```

Go still requires `Render` on `core.Component`. `template.Wrap` intercepts
`ViewComponent` and never calls the stub.

### 4. Wire the registries

```go
tmplReg, err := gouitemplate.NewRegistry(gouitemplate.Config{
    Root: filepath.Join(exampleDir, "views"),
})
if err != nil {
    log.Fatal(err)
}
defer tmplReg.Close()

coreReg := core.NewRegistry()
_ = coreReg.Register("counter", gouitemplate.Wrap(tmplReg, func() core.Component {
    return &Counter{}
}))
```

### 5. Verify

- `go build` your app
- Click through events; dirty tracking still uses `MarkDirty` / `ResetDirty`
  (`Wrap` clears dirty after a successful view render)

Reference implementation: [`examples/counter-view`](../../examples/counter-view)
(side by side with unchanged [`examples/counter`](../../examples/counter)).

## Checklist

- [ ] View file under `Config.Root` with matching dot-path from `View()`
- [ ] Factory registered with `template.Wrap`
- [ ] `StrictProps` enabled in production if you use `@props`
- [ ] `WatchForChanges` only in development
