# go-tinycore-template

A template for single-binary Go services on [go-kernel](https://github.com/KucherenkoIvan/go-kernel): DDD + hexagonal architecture in one static executable — embedded SQLite, commit-gated in-process events, graceful runtime, health endpoints. No containers, no external dependencies at runtime; the database is a file next to the binary.

Everything named **`changeme`** is deliberate scaffolding, not a domain: a nameless aggregate with a placeholder field and invariant, CRUD commands, CRUD events, and a CRUD sqlite repository. It exists so your first real feature is a rename-and-fill, not a blank page — and so the wiring (transactions, event publishing, migrations, composition root) is already correct and tested. There are **no API endpoints yet**; the HTTP server serves only `/livez` and `/healthz` until you add a transport adapter.

## Using the template

1. **Create your repo from it** (GitHub → "Use this template", or `gh repo create myapp --template KucherenkoIvan/go-tinycore-template`).
2. **Rename the module**: replace `github.com/KucherenkoIvan/go-tinycore-template` in `go.mod` and all imports, and update the depguard paths in `.golangci.yml`.
3. **go-kernel is a private module** — one-time setup:

	```sh
	go env -w GOPRIVATE=github.com/KucherenkoIvan/*
	git config --global url."git@github.com:KucherenkoIvan/".insteadOf "https://github.com/KucherenkoIvan/"
	```

4. `make test && make run` — then `curl localhost:8080/healthz`.
5. Build your first feature by renaming `changeme` (aggregate, events, migration, tests), then add its transport adapter (`adapters/rest/`) and register routes in `main.go`.

## Layout

```
cmd/app/main.go                     # composition root: config, storage, publisher, features, health, app.Run
internal/
  shared/infra/storage/             # embedded DB + migrations (schema source of truth)
  features/changeme/                # placeholder feature — rename or copy, then delete
    feature.go                      #   composition root: port -> adapter wiring
    domain/                         #   aggregate, CRUD events, domain errors
    application/ports/              #   repository + event-producer contracts
    application/usecases/managechangeme/  # create / update / delete commands
    adapters/sqlite/                #   CRUD repository
```

The architecture is documented in the kernel: [service structure](https://github.com/KucherenkoIvan/go-kernel/blob/master/docs/architecture/1-service-structure.md) → domain → application → infrastructure, plus per-package guides. `AGENTS.md` carries the operating rules for coding agents.

## Growing out of tiny

Everything sits behind ports, so scaling up is adapter swaps, not rewrites: sqlite → postgres (write the pg twin of `adapters/sqlite/`), in-process events → transactional outbox + Kafka (same producer port), REST and/or gRPC transports when endpoints appear, Redis when the in-memory cache stops being enough. The kernel guides cover each move.

## License

[0BSD](LICENSE).
