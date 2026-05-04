# AGENTS.md

## Architectural conventions

### Coding

- Don't use very short abbreviations for varia

### Layering

- **Handlers** (`internal/handlers`) are thin: decode JSON → call a service →
  map sentinel errors to HTTP status codes via `httpx.WriteError`. They must
  not contain business logic, validation, or DB access.
- **Services** (`internal/users`, `internal/presets`) own validation,
  authorization checks ("does this user own this group?"), and persistence.
  They expose `*Service` constructed via `NewService(...)` and called from
  `cmd/server/main.go` through `server.Deps`.
- **Models** (`internal/models`) are GORM entities only — no behavior beyond
  GORM hooks like `BeforeCreate`.

### Service inputs / outputs

For each service operation, define dedicated types in the package's `types.go`:

- Inputs: `XxxInput` structs with `normalize()` and `validate(...)` methods.
  - `normalize()` trims/lowercases as appropriate.
  - `validate()` returns wrapped sentinel errors:
    `fmt.Errorf("%w: human-readable reason", ErrValidation)`.
- Outputs: `XxxResult` for single-object writes, `XxxListItem` /
  `XxxItem` for reads. Do not return GORM models from services. CreatedAt and UpdatedAt should be the last fields in the struct

### Error handling

Each service package declares its own sentinel errors.

### HTTP helpers (`internal/httpx`)

Always go through these helpers — do not call `json.NewEncoder(w).Encode(...)`
or `chi.URLParam` directly from handlers. If there is no appropriate handle - create one.

### Database access

- Prefer the GORM v2 generic API: `gorm.G[models.X](s.db).Where(...).First(ctx)`
  / `Find(ctx)` / `Create(ctx, &x)` / `Update(ctx)` / `Delete(ctx)`.
- Use field helpers from `internal/generated` (e.g. `g.User.Email.Eq(...)`,
  `g.Preset.GroupId.Eq(...)`, `g.PresetGroup.Name.Set(...)`) instead of raw
  string column names. This keeps schema renames safe.
- For column lists (`Select`/`Omit`) use `g.X.Field.Column().Name`.
- Joins: see `presets.Service.ListPresets` for the canonical pattern using
  `clause.InnerJoin.Association(...)` and `g.X.Field.WithTable(alias)`.
- Treat `gorm.ErrRecordNotFound` explicitly and translate to `ErrNotFound` (or
  `ErrUnauthorized` for login). All other DB errors become `ErrInternal`; do
  not leak GORM errors out of services.
- Authorization: when mutating or reading user-scoped rows, **always** scope by
  `UserID` in the query (see `presets.Service.UpdateGroup`, `DeleteGroup`,
  etc.). For nested resources (presets within a group), verify the parent
  group belongs to the user before touching the child.

### Generated code

`internal/generated/` is produced by the gorm CLI from `internal/models/`. The
regen command (taken from a comment in `internal/handlers/auth.go`) is:

```bash
"$(go env GOPATH)/bin/gorm" gen -i ./internal/models -o ./internal/generated
```

## Adding a new endpoint — checklist

1. Add or extend the service in `internal/<domain>/`:
   - Define `XxxInput` with `normalize()` + `validate()`.
   - Add the method on `*Service`, returning typed results and sentinel errors.
2. Add the handler method in `internal/handlers/<domain>.go`:
   - Pull `userID` from context if auth-protected.
   - Call the service, branch on sentinel errors, write JSON.
3. Register the route in `internal/server/routes.go` under the right
   subrouter (public vs `s.requireAuth`).
4. If the route needs a new dependency, add it to `server.Deps` and wire it up
   in `cmd/server/main.go`.
5. Update `openapi.yaml` with the new path, request/response schemas, and
   error responses.
6. Run `go vet ./...` and `go test ./...`.

## Behavioral Guidelines

### Think Before Coding

Do not assume or hide confusion. Surface assumptions and tradeoffs before implementing.

- State assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them instead of choosing silently.
- If something is unclear, stop, name what is confusing, and ask.

### Simplicity First

Write the minimum code that solves the requested problem.

- Do not add features beyond what was asked.
- Do not add abstractions for single-use code.
- Do not add flexibility or configurability that was not requested.
- Do not add error handling for impossible scenarios.
- If a change is becoming much larger than necessary, simplify before continuing.

## Things not to do

- Do not edit anything under `internal/generated/`.
- Do not return raw GORM models or `gorm.ErrRecordNotFound` from services.
- Do not call `json.Encode` / `chi.URLParam` directly in handlers — use
  `httpx`.
- Do not bypass `userID` scoping on user-owned tables.
