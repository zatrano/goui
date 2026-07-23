# 10. Diffing Internals

Every re-render in GoUI goes through the same pipeline: render new HTML,
parse it into a small tree, diff it against the tree from the previous
render, and ship the resulting list of patches over the WebSocket. This
document explains that tree, the patch format, and the diff algorithm well
enough that you can reason about why a given change produces the patches it
does — and where to be careful.

Module path used throughout this document: `github.com/zatrano/goui`. The
relevant package is `diff` (`diff/node.go`, `diff/diff.go`, `diff/patch.go`,
`diff/serialize.go`).

## 1. `Node`

```go
// diff/node.go
type Node struct {
	Tag      string
	Text     string
	Attrs    map[string]string
	Key      string
	Children []*Node
}
```

- **Element nodes** have a non-empty `Tag` (e.g. `"div"`, `"li"`) and
  `Attrs`.
- **Text nodes** have `Tag == ""` (`isTextNode` checks exactly this) and
  carry their content in `Text`. Whitespace-only text nodes are dropped
  entirely while parsing (`convertHTMLNode` skips them), so pretty-printed
  Go string literals with indentation don't create phantom text nodes.
- **`Key`** is populated automatically from a `data-key` attribute, if
  present, while parsing:

  ```go
  for _, attr := range n.Attr {
      node.Attrs[attr.Key] = attr.Val
      if attr.Key == "data-key" {
          node.Key = attr.Val
      }
  }
  ```

  `data-key` stays in `Attrs` too (so it's still a normal HTML attribute in
  the DOM) — `Key` is just a diff-time convenience field derived from it.

`diff.ParseHTML(htmlStr string) (*Node, error)` parses a fragment (using
`golang.org/x/net/html`'s fragment parser with a `<div>` context) into a
synthetic `Tag: "root"` node whose `Children` are the actual top-level
nodes. `diff.Serialize(node *Node) string` does the inverse — it walks the
tree and writes back HTML, sorting attributes alphabetically for
deterministic output (used both for full-node HTML in patches and for
tests).

## 2. `Patch` and `PatchOp`

```go
// diff/patch.go
type PatchOp string

const (
	OpReplace    PatchOp = "replace"
	OpUpdateText PatchOp = "update_text"
	OpSetAttr    PatchOp = "set_attr"
	OpRemoveAttr PatchOp = "remove_attr"
	OpInsert     PatchOp = "insert"
	OpRemove     PatchOp = "remove"
	OpMove       PatchOp = "move"
)

type Patch struct {
	Op      PatchOp `json:"op"`
	Path    []int   `json:"path"`
	Tag     string  `json:"tag,omitempty"`
	Text    string  `json:"text,omitempty"`
	Attr    string  `json:"attr,omitempty"`
	Value   string  `json:"value,omitempty"`
	HTML    string  `json:"html,omitempty"`
	Key     string  `json:"key,omitempty"`
	FromIdx int     `json:"from_idx,omitempty"`
	ToIdx   int     `json:"to_idx,omitempty"`
}
```

`Patch` is what actually crosses the wire (as JSON, inside a `"render"`
frame's `payload`). A single re-render can produce zero, one, or many
patches.

### 2.1 `Path`

`Path` is a list of **child indices**, not a CSS selector or a DOM API path.
It is walked from the *component's root element* — see §3 below — by
repeatedly taking the Nth "meaningful child" (an element, or a non-blank
text node; the client's `meaningfulChildren()` helper applies the exact same
filtering rule used server-side by the parser). `Path: []` always means "the
root itself."

### 2.2 The seven ops

| Op | Meaning | Relevant fields |
|---|---|---|
| `replace` | Swap the node at `Path` for freshly-serialized HTML. Used for the very first render of a component (`Path: []`) and whenever a node's tag changes. | `HTML`, `Tag`, `Key` |
| `update_text` | Change a text node's content in place (no re-parse of surrounding structure). | `Text` |
| `set_attr` | Set/overwrite one attribute on the element at `Path`. | `Attr`, `Value` |
| `remove_attr` | Remove one attribute from the element at `Path`. | `Attr` |
| `insert` | Insert a new child (given as HTML) at `Path`, where the last index in `Path` is the position among the parent's children. | `HTML`, `Tag`, `Key` |
| `remove` | Remove the child at `Path`. | `Key`, `Tag` |
| `move` | Reposition an existing keyed child within its parent (parent identified by `Path`) from `FromIdx` to `ToIdx`, without touching its HTML. | `Key`, `FromIdx`, `ToIdx` |

The client's patch applier (`applyPatch` in `client/goui.js`) implements each
of these exactly as described, including boolean-property syncing for
`set_attr`/`remove_attr` on `checked`/`selected`/`disabled`/`readOnly` (see
[14-troubleshooting.md](14-troubleshooting.md) for why that matters), and it
also checks `data-goui-ignore` before applying anything under an ignored
subtree — see §6 and
[11-file-uploads.md](11-file-uploads.md)/rich-text notes.

## 3. Paths are relative to the component root, not the page

Two things cooperate to make `Path` always relative to a single component,
regardless of where that component is mounted in the page:

1. **`decorateComponentHTML`** (`ws/session.go`) tags whatever `Render()`
   returned with `data-goui-component="<id>"`:
   - If `Render()` returned exactly **one** root element, that element
     itself gets the `data-goui-component` attribute — no wrapper is added.
   - If it returned **zero** children (empty string), a placeholder
     `<div data-goui-component="<id>"></div>` is used.
   - If it returned **multiple sibling** root elements, they get wrapped in
     a synthetic `<div data-goui-component="<id>">...</div>`.

2. **`parseComponentTree`** then unwraps the synthetic `Tag: "root"` node
   that `ParseHTML` always produces, and — when there's exactly one
   top-level child — uses *that child* (the actual `data-goui-component`
   element) as the diff root, instead of the `"root"` wrapper:

   ```go
   // ws/session.go
   func parseComponentTree(html string) (*diff.Node, error) {
       tree, err := diff.ParseHTML(html)
       if err != nil {
           return nil, err
       }
       if len(tree.Children) == 1 {
           return tree.Children[0], nil
       }
       return tree, nil
   }
   ```

The consequence: **`Render()` should return exactly one root element.** If
it does, that element is both the tree root used for diffing *and* the
`[data-goui-component]` element the client looks up in the DOM
(`this.mount.querySelector('[data-goui-component="..."]')`), so `Path: [0]`
means "my first child," never "the second top-level thing on the page" and
never "the wrapper div GoUI added around my markup." If `Render()` returns
multiple siblings, GoUI will still work (via the synthetic wrapper), but you
gain one extra, otherwise pointless, `<div>` in the DOM and in every path
calculation — harmless, but worth avoiding by wrapping multi-element output
in your own single root tag instead of relying on the fallback.

## 4. The diff algorithm

`diff.Diff(old, new *Node) []Patch` walks both trees in lock-step, starting
at `path = []`  (`diffNode` in `diff/diff.go`):

1. **Nil cases** — `old == nil && new != nil` → `insert`; `old != nil && new
   == nil` → `remove`; both nil → nothing.
2. **Tag mismatch** — different `Tag` at the same position → `replace` the
   whole node (this also covers element-vs-text-node swaps, since a text
   node's `Tag` is always `""`).
3. **Both text nodes** — compare `Text`; emit `update_text` only if it
   actually changed.
4. **Same-tag elements** — diff attributes, then diff children.

### 4.1 Attribute diffing

`diffAttrs` computes the **union** of old and new attribute keys, sorted for
deterministic patch ordering, and for each key:

- present only in `new` → `set_attr`
- present only in `old` → `remove_attr`
- present in both but different value → `set_attr`
- present in both, same value → no patch

### 4.2 Child diffing: indexed vs. keyed

`diffChildren` picks one of two strategies **per parent node**, based on
whether *either* side's children contain any `data-key`:

```go
func diffChildren(oldChildren, newChildren []*Node, path []int, patches *[]Patch) {
	if hasAnyKey(oldChildren) || hasAnyKey(newChildren) {
		diffKeyedChildren(oldChildren, newChildren, path, patches)
		return
	}
	diffIndexedChildren(oldChildren, newChildren, path, patches)
}
```

**Indexed diffing** (no keys anywhere in this child list) is purely
positional: diff the common prefix index-by-index (recursing into each
pair), then `insert` any extra new tail children, then `remove` any extra
old tail children (from the end backwards, so earlier indices stay valid).
This is efficient and correct for lists that only grow/shrink at the end,
but for a list that's reordered *in the middle*, indexed diffing degenerates
into "replace/update everything from the first differing index onward" —
which is exactly why keyed lists exist.

**Keyed diffing** — see §5.

## 5. Keyed lists: a key→position map, not an LCS

The task of matching "this item used to be at position 3 and is now at
position 0" optimally (minimum patch count for arbitrary reorders) is
classically solved with a longest-common-subsequence algorithm. GoUI does
**not** implement LCS. `diffKeyedChildren` (`diff/diff.go`) instead:

1. Builds four small maps up front: `oldByKey`/`oldPos` and
   `newByKey`/`newPos`, from `data-key` to node / to index, for whichever
   children actually have a key. Unkeyed children in a keyed list are
   diffed positionally, inline, alongside the keyed logic.
2. Walks `oldChildren` **in reverse** and emits a `remove` patch for any
   keyed child whose key no longer exists in `newChildren`.
3. Determines a single boolean, `hasInsertOrRemove`: true if any key exists
   in one side but not the other (i.e. the list actually gained or lost
   members, not just reordered existing ones).
4. Walks `newChildren` **forward**:
   - Unkeyed new child → diff positionally against whatever's at that same
     index in the old list (or `insert` if nothing was there).
   - Keyed new child with no matching old key → `insert`.
   - Keyed new child with a matching old key:
     - If `!hasInsertOrRemove` **and** its position changed
       (`oldPos[key] != newPos[key]`), emit a `move` patch
       (`FromIdx: oldPos[key], ToIdx: newPos[key]`).
     - Always recurse into `diffNode(oldChild, newChild, ...)` too, so
       content/attribute changes on a moved (or stationary) keyed item are
       still captured.

In short: **this is a key→position lookup, not a subsequence algorithm.**
Every changed position for an existing key becomes its own independent
`move` patch, computed from the *original* before/after position maps (not
recomputed as earlier moves are applied). For the common cases — appending,
prepending, removing an item, or moving a single item — this produces
exactly the patches you'd expect. For **simultaneous multi-item reorders**
(e.g. swapping several pairs at once, or a full shuffle), the emitted `move`
patches are applied sequentially against a live DOM whose child order
changes after each one; because indices are derived from snapshots taken
before any move is applied, a sequence of several simultaneous moves is not
guaranteed to reduce to a minimal (or even obviously correct-looking)
sequence of DOM operations the way a proper LCS-based reconciler's would.
Test this specifically if your keyed lists support drag-to-reorder or
multi-select-and-move UI, and prefer changing one position at a time (e.g.
"move item up/down by one" controls) or emitting a full `replace` of the
parent list rather than relying on many concurrent `move` patches from a
single render if you need bullet-proof correctness on heavy reordering.

## 6. `data-goui-ignore`

Independent of keys, any subtree marked `data-goui-ignore` is skipped by the
**client** when applying patches — see `isGoUIIgnored`/`applyPatch` in
`client/goui.js`. This exists for client-owned widgets (Quill, CodeMirror)
whose DOM must never be touched by the reconciler; combine it with
`core.ErrSkipRender` server-side so the server doesn't even bother computing
a patch set for events that only mirror state into those widgets. See
[14-troubleshooting.md](14-troubleshooting.md) for the full pattern.

## 7. `data-key` for dynamic lists — practical guidance

Add `data-key="<stable-id>"` to the repeated root element of any list you
render that can be reordered, filtered, or spliced in the middle — not to
lists that only ever grow/shrink at one end:

```go
var b strings.Builder
b.WriteString("<ul>")
for _, item := range items {
	b.WriteString(`<li data-key="` + html.EscapeString(item.ID) + `">`)
	b.WriteString(html.EscapeString(item.Label))
	b.WriteString("</li>")
}
b.WriteString("</ul>")
```

Rules of thumb:

- **The key must be stable across renders** for the same logical item (a
  database ID, not a slice index — a slice index defeats the entire purpose,
  since it's exactly what changes on reorder).
- **Either all or none** of a given child list should carry keys in
  practice; `hasAnyKey` triggers keyed mode for the *whole* sibling list the
  moment even one child has a `data-key`, so a stray key on one `<li>` in an
  otherwise-unkeyed list forces the (slower, move-aware) keyed path for all
  of them.
- **Keys only affect that one parent's direct children.** Nested lists need
  their own `data-key` on their own items; keys don't propagate.
- Un-keyed lists that only grow at the tail (chat messages, activity feeds,
  "load more" pagination) are already optimal under indexed diffing —
  don't add keys there, it adds attribute noise for no patch-count benefit.

## 8. Performance notes

- **Attribute diffing is O(keys) per node**, with a sort for deterministic
  ordering — negligible even for elements with many attributes.
- **Indexed child diffing is O(min(old, new))** for the shared prefix, plus
  O(|Δlength|) for the tail insert/remove — cheap, and this is the default
  path for the overwhelming majority of markup (most elements don't have
  keyed children at all).
- **Keyed child diffing is O(n)** to build the maps plus O(n) to walk — no
  quadratic blowup — but produces `move` patches whose *number* can be up to
  O(n) in the worst case (every item's position changed), each of which the
  client applies as its own DOM `insertBefore` call. A full-list `replace`
  is one HTML string and one DOM swap; a heavily-reordered keyed list can
  turn into dozens of small ops. For very large lists (hundreds+ of rows)
  that reorder frequently, measure both and consider whether a coarser
  `replace` of the whole list (drop the keys, or force a full re-render) is
  actually cheaper than many `move` patches — GoUI does not decide this for
  you.
- **`Serialize` is only called for `replace`/`insert` payload HTML** — nodes
  that only need `set_attr`/`update_text`/`remove_attr`/`move` never pay the
  cost of re-serializing their subtree, which is the main reason to prefer
  narrow updates (e.g. a text-only counter) over wrapping frequently-changing
  content in an element whose *tag* changes, which forces a full `replace`.
- **The whole pipeline runs once per `HandleEvent` call that doesn't return
  `core.ErrSkipRender`.** Debounce noisy client events (`g-debounce`, used
  throughout `forms/*`) so a fast typist doesn't trigger a parse+diff+patch
  cycle on every keystroke — see the `TextInput`/`RichTextEditor` field
  implementations for the existing convention (typically 100–350ms).
