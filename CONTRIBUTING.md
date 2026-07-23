# Contributing to GoUI

Thanks for considering a contribution. GoUI is a Go-first, server-driven UI framework; keep that model intact.

## Ground rules

1. **Do not break existing public behavior** unless the change is a clear bug fix with tests.
2. **Tests are mandatory** for behavioral changes: `go test ./...` must stay green.
3. **No new heavy dependencies in the core module** without discussion. Prefer the Go standard library. Framework-specific code belongs in nested `adapters/*` modules.
4. **No Docker requirement** for core development or examples.
5. Documentation lives in `docs/en` and `docs/tr` — update **both** languages when you change public APIs.

## Setup

```bash
git clone https://github.com/zatrano/goui.git
cd goui
# go.work wires the core module, adapters, and examples
go test ./...
go test ./adapters/stdlib ./adapters/fiber ./adapters/gin ./adapters/echo
go run ./examples/counter            # Fiber demo :3000
go run ./examples/adapters/nethttp   # stdlib demo :3010
```

## Code style

- Idiomatic Go; `gofmt` / `go vet` clean
- Match existing package patterns (`core`, `ws`, `forms`, `adapters/…`)
- Prefer small, focused PRs over large mixed changes
- Client JS: vanilla ES modules, no bundler in core
- Keep the root module free of Fiber/Gin/Echo/Chi direct dependencies

## Pull requests

1. Open an issue for non-trivial features (optional but appreciated)
2. Branch from `main`
3. Add/adjust tests and docs
4. Ensure:

```bash
go build ./...
go vet ./...
go test ./...
```

5. Describe *why* in the PR body; link issues

## Large features

Propose in an issue first. New form controls and alternative storage backends (S3/MinIO) are welcome as scoped PRs implementing existing interfaces where possible.

## License

By contributing, you agree your contributions are licensed under the project LICENSE (MIT draft until finalized).
