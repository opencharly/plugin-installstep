# AGENTS.md — plugin-installstep

Standalone plugin repo for the compiler-emitted builtin `InstallStep` kinds
(`class:step`). The plugin is a Go module at `candy/plugin-installstep/` (module
path `github.com/opencharly/plugin-installstep/candy/plugin-installstep`); the
root `charly.yml` only declares `discover: candy` so the repo is a project and
its candy is scanned.

Canonical files:

- `candy/plugin-installstep/charly.yml` — the `plugin-installstep:` candy entity
  (`plugin:` block, `plan:` check).
- `candy/plugin-installstep/` — the Go source: `plugin.go`, `oci_dispatch.go`,
  `schema/installstep.cue`, `cmd/serve/main.go`.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-internals:install-plan` — the InstallPlan IR, the deploy lifecycle
  (`OpExecute` reverse channel, ledger record) and the `OpEmit` build-time
  fragment. Load before changing a step body.
- `/charly-build:generate` · `/charly-internals:generate-source` — the
  build-time step emission pipeline (`OpEmit` → fragment).
- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the unified Provider model, the `step` provider class.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-installstep/` — compile the plugin module.
- `go test ./...` in `candy/plugin-installstep/` — the plugin's Go tests.
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.
- The changed path is exercised by every box build (the build-emit leg) and
  every deploy that walks an InstallStep.

## Modify this repo

- Edit the `plugin-installstep:` candy entity, the Go source, and
  `schema/installstep.cue` **together**.
- There is NO authored `plugin_input` for these steps (they are compiler-emitted
  from declarative candy fields); the shipped CUE schema is vestigial and exists
  only to satisfy the plugin load gate. Do not invent an `InputDef`.
- The `OpEmit` payload is a `spec.InstallStepView`; keep the pure and
  host-coupled split intact when adding a word.

## Landing

- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Load
  `/charly-internals:git-workflow` before any git/PR action; history lives in
  `CHANGELOG/`. Do not restate its rules here.
