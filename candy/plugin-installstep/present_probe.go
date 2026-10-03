package installstep

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/opencharly/sdk"
	"github.com/opencharly/sdk/buildkit"
	spec "github.com/opencharly/spec/spec"
)

// systemPackagesTeardownDelta returns the subset of the DECLARED package names the
// venue did NOT already have. declared keeps its declared form (e.g. "curl=8.0")
// because the install template renders that same raw form; the probe's output is
// compared AFTER buildkit.PkgName — the ONE canonical version-strip (R3), the same
// normalizer the uninstall template uses via the pkgName template func.
//
// The result is ALWAYS non-nil. An empty result is authoritative and means every
// declared package was already present, which Reverse() renders as NO removal op
// at all — so never return nil here to mean "nothing": nil means "not determined".
func systemPackagesTeardownDelta(declared []string, presentStdout string) []string {
	present := make(map[string]bool)
	for _, line := range strings.Split(presentStdout, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			present[name] = true
		}
	}
	delta := make([]string, 0, len(declared))
	for _, p := range declared {
		if !present[buildkit.PkgName(p)] {
			delta = append(delta, p)
		}
	}
	return delta
}

// probePresentPackages renders the format's present_template and runs it on the
// venue, returning the probe's stdout. ok=false means the venue did NOT answer —
// absent deps/distro config, not an install-phase step, no packages, a dry run, no
// format def, no present_template configured, a render failure, or a failed or
// non-zero run — and the caller MUST then leave SystemPackagesStep.Installed nil so
// the prior declared-list behaviour is kept. It must NEVER report an empty answer
// as success: empty is authoritative and would record no removal at all.
func probePresentPackages(ctx context.Context, deps *sdk.HostStepDeps, st *spec.SystemPackagesStep) (string, bool) {
	if deps == nil || deps.Exec == nil || deps.DistroCfg == nil {
		return "", false
	}
	if st.Phase != spec.PhaseInstall || len(st.Packages) == 0 {
		return "", false
	}
	if deps.Opts.DryRun {
		fmt.Fprintf(os.Stderr, "plugin-installstep: dry run — not probing %s for already-present packages; the teardown keeps the declared set\n", st.Format)
		return "", false
	}
	fd := deps.DistroCfg.FindFormat(st.Format)
	if fd == nil {
		fmt.Fprintf(os.Stderr, "plugin-installstep: no format %q in the distro config — cannot probe already-present packages; the teardown keeps the declared set\n", st.Format)
		return "", false
	}
	if strings.TrimSpace(fd.PresentTemplate) == "" {
		fmt.Fprintf(os.Stderr, "plugin-installstep: format %q declares no present_template — cannot compute the teardown delta; the teardown keeps the declared set\n", st.Format)
		return "", false
	}
	ictx := buildkit.NewInstallContext(st.RawInstallContext, fd.CacheMount)
	cmd, err := buildkit.RenderTemplate(st.Format+"-present", fd.PresentTemplate, ictx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "plugin-installstep: rendering %s present_template failed: %v; the teardown keeps the declared set\n", st.Format, err)
		return "", false
	}
	if cmd = strings.TrimSpace(cmd); cmd == "" {
		return "", false
	}
	stdout, stderr, exit, err := deps.Exec.RunCapture(ctx, cmd)
	if err != nil || exit != 0 {
		fmt.Fprintf(os.Stderr, "plugin-installstep: probing %s for already-present packages failed (exit=%d err=%v stderr=%q); the teardown keeps the declared set\n", st.Format, exit, err, strings.TrimSpace(stderr))
		return "", false
	}
	return stdout, true
}
