# plugin-installstep

The dual-placement `class:step` plugin for [opencharly/charly](https://github.com/opencharly/charly) —
it serves the build-context `OpEmit` leg (and the deploy-context `OpExecute` leg)
for the compiler-emitted builtin `InstallStep` kinds.

Every `InstallStep` kind is plugin-served. This candy owns twelve of them; the
`ExternalPlugin` kind dispatches through its own `class:step` plugin.

## What it provides

| Capability | Surface |
|---|---|
| `step:file`, `step:shell-hook`, `step:shell-snippet` | pure Containerfile-fragment renderers |
| `step:service-packaged`, `step:service-custom`, `step:repo-change` | pure Containerfile-fragment renderers |
| `step:apk-install`, `step:reboot`, `step:extract` | declared `Emits=false` (no build fragment) |
| `step:system-packages`, `step:builder`, `step:local-pkg-install`, `step:op` | host-coupled `OpEmit` renderers |
| `step:oci-dispatch` | the full core provider-registry dispatch for ANY step kind |

## The two legs

- **PURE kinds** render their fragment by string formatting from the
  compiler-produced `spec.InstallStepView`. `apk-install` and `reboot` are the
  no-op-emit members (an image build installs no apk / reboots nothing).
- **HOST-COUPLED kinds** (`system-packages`, `builder`, `local-pkg-install`,
  `op`) render directly against the resolved-project envelope — fetched ONCE per
  project dir via `InvokeProvider("build","project")` and cached.
- **`oci-dispatch`** decides which peer `class:step`/`class:verb` provider
  renders a pod-overlay fragment and dispatches to it via the generic
  reverse-channel `DescribeProvider` + `InvokeProvider` legs.

The DEPLOY leg for all these kinds stays in `sdk/kit.WalkPlans` (rendered over
the executor reverse channel); this plugin serves `OpEmit` (the build-emit the
host's `deploykit.OCITarget` splices).

## How to use it

The plugin is compiled into charly (`compiled_plugins:`), so no candy
composition is needed. The kinds are emitted from declarative candy fields —
there is no authored `plugin:` step.

## Layout

- `candy/plugin-installstep/` — the plugin module: `plugin.go`, `oci_dispatch.go`,
  `schema/installstep.cue`, `cmd/serve/main.go`.
- `candy/plugin-installstep/charly.yml` — the `plugin-installstep:` candy entity.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skill: `/charly-internals:install-plan` — the InstallPlan IR, the
  `pluginDeployTarget` deploy lifecycle, and the `OpEmit` build-time fragment.
  This candy carries no `skill:` entity of its own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-build:generate` · `/charly-internals:generate-source` — the
  build-time step emission pipeline.
- `/charly-internals:plugin` — the plugin/provider model, including the `step`
  provider class.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.
