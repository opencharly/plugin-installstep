package installstep

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
	"github.com/opencharly/spec/spec"
)

// TestSystemPackagesTeardownDelta pins the SUBTRACTION SPACE (charly#771): the pac
// install template renders the RAW declared names (`curl=8.0`) while the probe's
// stdout carries the STRIPPED ones (`curl`), so the comparison MUST go through
// buildkit.PkgName. A direct string comparison would silently leave `curl=8.0` in
// the delta — the full declared list, i.e. the bug unchanged.
func TestSystemPackagesTeardownDelta(t *testing.T) {
	cases := []struct {
		name     string
		declared []string
		stdout   string
		want     []string
	}{
		{
			name:     "stripped probe subtracts the raw declared pin (the trap case)",
			declared: []string{"curl=8.0", "helm", "kubectl"},
			stdout:   "curl\n",
			want:     []string{"helm", "kubectl"},
		},
		{
			name:     "every declared package already present yields an empty NON-NIL delta",
			declared: []string{"curl=8.0"},
			stdout:   "curl\n",
			want:     []string{},
		},
		{
			name:     "no probe output keeps every declared package",
			declared: []string{"helm"},
			stdout:   "",
			want:     []string{"helm"},
		},
		{
			name:     "blank lines and surrounding whitespace are ignored",
			declared: []string{"helm", "kubectl"},
			stdout:   "  helm  \n\n\nkubectl\n\n",
			want:     []string{},
		},
		{
			name:     "an unrelated present package subtracts nothing",
			declared: []string{"helm"},
			stdout:   "curl\n",
			want:     []string{"helm"},
		},
		{
			name:     "a nil declared set yields an empty NON-NIL delta",
			declared: nil,
			stdout:   "curl\n",
			want:     []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := systemPackagesTeardownDelta(tc.declared, tc.stdout)
			if got == nil {
				t.Fatal("delta must be non-nil: nil means not-determined, and Reverse() would fall back to the full declared list")
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("delta = %v, want %v", got, tc.want)
			}
		})
	}
}

// fakeDeployExecutor is a capture-only spec.DeployExecutor double: RunCapture answers
// with a canned stdout (recording the probe script it was asked to run) and RunSystem
// records the install command it was asked to run. No venue, no network.
type fakeDeployExecutor struct {
	captureStdout string
	ranCapture    string
	ranSystem     string
}

var _ spec.DeployExecutor = (*fakeDeployExecutor)(nil)

func (f *fakeDeployExecutor) Venue() string { return "fake" }

func (f *fakeDeployExecutor) RunSystem(_ context.Context, script string, _ spec.EmitOpts) error {
	f.ranSystem = script
	return nil
}

func (f *fakeDeployExecutor) RunUser(context.Context, string, spec.EmitOpts) error { return nil }

func (f *fakeDeployExecutor) RunBuilder(context.Context, spec.BuilderRunOpts) ([]byte, error) {
	return nil, nil
}

func (f *fakeDeployExecutor) PutFile(context.Context, string, string, uint32, bool, spec.EmitOpts) error {
	return nil
}

func (f *fakeDeployExecutor) GetFile(context.Context, string, bool, spec.EmitOpts) ([]byte, error) {
	return nil, nil
}

func (f *fakeDeployExecutor) RunCapture(_ context.Context, script string) (string, string, int, error) {
	f.ranCapture = script
	return f.captureStdout, "", 0, nil
}

func (f *fakeDeployExecutor) RunInteractive(context.Context, string) (int, error) { return 0, nil }
func (f *fakeDeployExecutor) RunStream(context.Context, string) (int, error)      { return 0, nil }
func (f *fakeDeployExecutor) Kind() string                                        { return "fake" }

func (f *fakeDeployExecutor) ResolveHome(context.Context, string) (string, error) {
	return "/root", nil
}

// TestExecDeployStepSystemPackagesDelta proves the probed delta reaches
// ReverseOps.Targets through the real execDeployStep wiring: the probe's stdout is
// subtracted (in the DECLARED form for the survivors) from the declared packages,
// and an all-present probe records NO removal op at all.
func TestExecDeployStepSystemPackagesDelta(t *testing.T) {
	cfg := &spec.DistroConfig{Distro: map[string]*spec.ResolvedDistro{
		"test": {Format: map[string]*spec.Format{"pac": {PresentTemplate: "probe"}}},
	}}

	invoke := func(t *testing.T, probeStdout string) (spec.DeployReply, *fakeDeployExecutor) {
		t.Helper()
		fake := &fakeDeployExecutor{captureStdout: probeStdout}
		deps := &sdk.HostStepDeps{Exec: fake, DistroCfg: cfg, Opts: spec.EmitOpts{}}
		ctx := sdk.ContextWithHostStepDeps(context.Background(), deps)

		step := &spec.SystemPackagesStep{
			Format:   "pac",
			Phase:    spec.PhaseInstall,
			Packages: []string{"curl=8.0", "helm", "kubectl"},
		}
		pj, err := json.Marshal(spec.StepToView(step))
		if err != nil {
			t.Fatalf("marshal InstallStepView: %v", err)
		}
		reply, err := execDeployStep(ctx, &pb.InvokeRequest{Reserved: "system-packages", Op: opExecute, ParamsJson: pj})
		if err != nil {
			t.Fatalf("execDeployStep: %v", err)
		}
		var dr spec.DeployReply
		if err := json.Unmarshal([]byte(reply.GetResultJson()), &dr); err != nil {
			t.Fatalf("decode DeployReply: %v", err)
		}
		return dr, fake
	}

	t.Run("already-present packages are subtracted, survivors keep declared form", func(t *testing.T) {
		dr, fake := invoke(t, "curl\n")
		if fake.ranCapture != "probe" {
			t.Fatalf("RunCapture script = %q, want %q (the rendered present_template)", fake.ranCapture, "probe")
		}
		if len(dr.ReverseOps) != 1 {
			t.Fatalf("ReverseOps = %+v, want exactly one package-remove op", dr.ReverseOps)
		}
		op := dr.ReverseOps[0]
		if op.Kind != spec.ReverseOpPackageRemove {
			t.Errorf("Kind = %q, want %q", op.Kind, spec.ReverseOpPackageRemove)
		}
		if op.Format != "pac" {
			t.Errorf("Format = %q, want pac", op.Format)
		}
		if want := []string{"helm", "kubectl"}; !slices.Equal(op.Targets, want) {
			t.Errorf("Targets = %v, want %v (the declared pin curl=8.0 must be subtracted; survivors keep the DECLARED form)", op.Targets, want)
		}
	})

	t.Run("every declared package already present records no removal at all", func(t *testing.T) {
		dr, _ := invoke(t, "curl\nhelm\nkubectl\n")
		if len(dr.ReverseOps) != 0 {
			t.Fatalf("ReverseOps = %+v, want none (a non-nil EMPTY Installed set is authoritative)", dr.ReverseOps)
		}
	})
}
