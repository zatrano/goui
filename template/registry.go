package template

import (
	"bytes"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

const defaultRoot = "./views"
const gouiExt = ".goui.html"

// Config configures NewRegistry.
type Config struct {
	// Root is the directory scanned for .goui.html files. Default: "./views".
	Root string
	// ExtraFuncs are merged on top of BaseFuncMap().
	ExtraFuncs htmltemplate.FuncMap
	// StrictProps enables @props field-name checks at compile time.
	// Default false for easier development; prefer true in production.
	StrictProps bool
	// WatchForChanges enables fsnotify-based hot reload (dev only; keep false in prod).
	WatchForChanges bool
	// OnReload is called after a successful hot-reload recompile.
	// Wire this to ws.Hub.Broadcast — the template package does not import ws.
	OnReload func()
	// OnReloadError is called when a hot-reload recompile fails.
	// If nil, errors are logged with log.Printf.
	OnReloadError func(error)
}

// Registry holds a compiled, in-memory template set.
// All fields are private; access is via Render / Exists / Close / Warnings.
type Registry struct {
	mu       sync.RWMutex
	root     *htmltemplate.Template
	names    map[string]struct{}
	warnings []string

	cfg       Config
	rootDir   string
	watcher   *fsnotify.Watcher
	stopCh    chan struct{}
	doneCh    chan struct{}
	closeOnce sync.Once
}

// NewRegistry scans cfg.Root, compiles every .goui.html file, and returns a
// ready-to-render Registry. Compilation errors from all files are joined.
func NewRegistry(cfg Config) (*Registry, error) {
	rootDir := cfg.Root
	if rootDir == "" {
		rootDir = defaultRoot
	}

	reg := &Registry{
		cfg:     cfg,
		rootDir: rootDir,
	}

	root, names, warnings, err := reg.build()
	if err != nil {
		return nil, err
	}
	reg.root = root
	reg.names = names
	reg.warnings = warnings

	if cfg.WatchForChanges {
		if err := reg.startWatch(); err != nil {
			return nil, err
		}
	}
	return reg, nil
}

// Warnings returns soft diagnostics from StrictProps (unused @props fields).
func (r *Registry) Warnings() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.warnings))
	copy(out, r.warnings)
	return out
}

// build compiles all templates under rootDir into a new template set.
// Does not mutate r.root / r.names (caller swaps under lock on reload).
func (r *Registry) build() (*htmltemplate.Template, map[string]struct{}, []string, error) {
	units, files, parseErrs := compileAll(r.rootDir)
	var errs []error
	errs = append(errs, parseErrs...)

	if cycleErr := detectExtendsCycles(units); cycleErr != nil {
		errs = append(errs, cycleErr)
	}
	errs = append(errs, validateDeps(units)...)

	var warnings []string
	if r.cfg.StrictProps {
		propErrs, warns := checkStrictProps(files)
		errs = append(errs, propErrs...)
		warnings = warns
	}

	if len(errs) > 0 {
		return nil, nil, nil, errors.Join(errs...)
	}

	exists := func(name string) bool {
		_, ok := units[name]
		return ok
	}

	names := make(map[string]struct{}, len(units))
	for name := range units {
		names[name] = struct{}{}
	}

	funcs := r.bindFuncs(r.cfg.ExtraFuncs)
	root := htmltemplate.New("").Funcs(funcs)

	var standalone []string
	var extending []string
	for name, u := range units {
		if u.Extends == "" {
			standalone = append(standalone, name)
		} else {
			extending = append(extending, name)
		}
	}
	sort.Strings(standalone)
	sort.Strings(extending)

	for _, name := range standalone {
		u := units[name]
		src := resolveIncludeIf(u.Body, exists)
		if _, err := root.New(name).Parse(src); err != nil {
			errs = append(errs, fmt.Errorf("%s: parse: %w", name, err))
		}
	}

	for _, name := range extending {
		body, defs, err := buildExtendsParts(name, units, exists)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if _, err := root.New(name).Parse(body); err != nil {
			errs = append(errs, fmt.Errorf("%s: parse body: %w", name, err))
			continue
		}
		if defs != "" {
			secName := name + ".__sections"
			if _, err := root.New(secName).Parse(defs); err != nil {
				errs = append(errs, fmt.Errorf("%s: parse sections: %w", name, err))
			}
		}
	}

	for _, u := range units {
		for defName, body := range u.SlotDefines {
			src := resolveIncludeIf(body, exists)
			if _, err := root.New(defName).Parse(src); err != nil {
				errs = append(errs, fmt.Errorf("slot %s: parse: %w", defName, err))
			}
		}
	}

	if len(errs) > 0 {
		return nil, nil, nil, errors.Join(errs...)
	}
	return root, names, warnings, nil
}

// Render executes the named template (dot-path) with data and returns HTML.
// No filesystem I/O is performed (production / default mode).
func (r *Registry) Render(name string, data any) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.names[name]; !ok {
		return "", fmt.Errorf("template %q not found", name)
	}
	var buf bytes.Buffer
	if err := r.root.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Exists reports whether a template with the given dot-path was compiled.
func (r *Registry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.names[name]
	return ok
}

func (r *Registry) bindFuncs(extra htmltemplate.FuncMap) htmltemplate.FuncMap {
	funcs := htmltemplate.FuncMap{}
	for k, v := range BaseFuncMap() {
		funcs[k] = v
	}
	funcs["component"] = r.componentFn
	for k, v := range extra {
		funcs[k] = v
	}
	return funcs
}

// componentFn renders named/default slots in the caller's context, then executes
// the component template with a Dot{Props, Slots, DefaultSlot}.
// Must not lock r.mu — called during Render while RLock is already held.
func (r *Registry) componentFn(target string, props any, callerDot any, slotArgs ...string) (htmltemplate.HTML, error) {
	if len(slotArgs)%2 != 0 {
		return "", fmt.Errorf("component: slotArgs must be pairs, got %d", len(slotArgs))
	}
	dot := &Dot{
		Props: props,
		Slots: make(map[string]htmltemplate.HTML),
	}
	for i := 0; i+1 < len(slotArgs); i += 2 {
		slotName, defineName := slotArgs[i], slotArgs[i+1]
		var buf bytes.Buffer
		if err := r.root.ExecuteTemplate(&buf, defineName, callerDot); err != nil {
			return "", fmt.Errorf("component %s: rendering slot %q: %w", target, slotName, err)
		}
		//nolint:gosec // G203: slot body already produced by html/template (escaped unless raw)
		html := htmltemplate.HTML(buf.String())
		if slotName == "" {
			dot.DefaultSlot = html
		} else {
			dot.Slots[slotName] = html
		}
	}
	var out bytes.Buffer
	if err := r.root.ExecuteTemplate(&out, target, dot); err != nil {
		return "", fmt.Errorf("component %s: %w", target, err)
	}
	//nolint:gosec // G203: component output already produced by html/template
	return htmltemplate.HTML(out.String()), nil
}

func compileAll(rootDir string) (map[string]*CompileUnit, map[string]*File, []error) {
	units := make(map[string]*CompileUnit)
	files := make(map[string]*File)
	var errs []error

	info, err := os.Stat(rootDir)
	if err != nil {
		return units, files, []error{fmt.Errorf("template root %q: %w", rootDir, err)}
	}
	if !info.IsDir() {
		return units, files, []error{fmt.Errorf("template root %q is not a directory", rootDir)}
	}

	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return units, files, []error{fmt.Errorf("template root %q: %w", rootDir, err)}
	}
	defer func() { _ = root.Close() }()

	_ = fs.WalkDir(root.FS(), ".", func(rel string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			errs = append(errs, walkErr)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(rel, gouiExt) {
			return nil
		}
		name := pathToDot(rel)
		src, err := root.ReadFile(rel)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			return nil
		}
		displayPath := filepath.Join(rootDir, filepath.FromSlash(rel))
		file, err := ParseSource(displayPath, src)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		unit, err := Generate(file)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		unit.Name = name
		if _, dup := units[name]; dup {
			errs = append(errs, fmt.Errorf("duplicate template name %q", name))
			return nil
		}
		units[name] = unit
		files[name] = file
		return nil
	})

	return units, files, errs
}

func pathToDot(rel string) string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, gouiExt)
	return strings.ReplaceAll(rel, "/", ".")
}

func validateDeps(units map[string]*CompileUnit) []error {
	var errs []error
	for name, u := range units {
		if u.Extends != "" {
			if _, ok := units[u.Extends]; !ok {
				errs = append(errs, fmt.Errorf("%s: @extends target %q not found", name, u.Extends))
			}
		}
		for _, inc := range u.Includes {
			if _, ok := units[inc]; !ok {
				errs = append(errs, fmt.Errorf("%s: @include target %q not found", name, inc))
			}
		}
		for _, c := range u.Components {
			if _, ok := units[c]; !ok {
				errs = append(errs, fmt.Errorf("%s: @component target %q not found", name, c))
			}
		}
	}
	return errs
}

func detectExtendsCycles(units map[string]*CompileUnit) error {
	var cycles []error
	for name := range units {
		path := []string{name}
		cur := name
		seen := map[string]bool{name: true}
		for {
			u := units[cur]
			if u == nil || u.Extends == "" {
				break
			}
			next := u.Extends
			if seen[next] {
				cycles = append(cycles, fmt.Errorf(
					"circular @extends chain: %s",
					strings.Join(append(path, next), " -> "),
				))
				break
			}
			if _, ok := units[next]; !ok {
				break // missing target reported by validateDeps
			}
			seen[next] = true
			path = append(path, next)
			cur = next
		}
	}
	if len(cycles) == 0 {
		return nil
	}
	return errors.Join(cycles...)
}

func resolveIncludeIf(src string, exists func(string) bool) string {
	for {
		start := strings.Index(src, IncludeIfMarkerPrefix)
		if start < 0 {
			return src
		}
		rest := src[start+len(IncludeIfMarkerPrefix):]
		end := strings.Index(rest, IncludeIfMarkerSuffix)
		if end < 0 {
			return src // malformed; leave as-is
		}
		target := rest[:end]
		repl := ""
		if exists(target) {
			repl = `{{template "` + target + `" .}}`
		}
		src = src[:start] + repl + rest[end+len(IncludeIfMarkerSuffix):]
	}
}

func buildExtendsParts(leaf string, units map[string]*CompileUnit, exists func(string) bool) (body, defs string, err error) {
	chain, err := extendsChain(leaf, units)
	if err != nil {
		return "", "", err
	}
	ns := leaf
	rootBody := resolveIncludeIf(chain[0].Body, exists)
	body = applyNamespace(rootBody, ns)

	var b strings.Builder
	for _, u := range chain {
		keys := make([]string, 0, len(u.Sections))
		for k := range u.Sections {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sec := resolveIncludeIf(u.Sections[k], exists)
			b.WriteString(applyNamespace(sec, ns))
		}
	}
	return body, b.String(), nil
}

func extendsChain(leaf string, units map[string]*CompileUnit) ([]*CompileUnit, error) {
	var chain []*CompileUnit
	cur := leaf
	seen := map[string]bool{}
	for {
		if seen[cur] {
			return nil, fmt.Errorf("circular @extends chain involving %q", leaf)
		}
		seen[cur] = true
		u, ok := units[cur]
		if !ok {
			return nil, fmt.Errorf("@extends chain broken: %q not found", cur)
		}
		chain = append(chain, u)
		if u.Extends == "" {
			break
		}
		cur = u.Extends
	}
	// chain is leaf→…→root; reverse to root→…→leaf
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

var namespaceRe = regexp.MustCompile(`\{\{(block|define)\s+"([^"]+)"`)

// applyNamespace prefixes block/define names so each @extends page has an
// isolated define namespace inside the shared template set.
func applyNamespace(src, ns string) string {
	return namespaceRe.ReplaceAllStringFunc(src, func(m string) string {
		sub := namespaceRe.FindStringSubmatch(m)
		if len(sub) != 3 {
			return m
		}
		return "{{" + sub[1] + " \"" + ns + "/" + sub[2] + "\""
	})
}
