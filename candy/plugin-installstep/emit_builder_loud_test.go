package installstep

import (
	"strings"
	"testing"

	"github.com/opencharly/sdk/buildkit"
	"github.com/opencharly/sdk/deploykit"
	"github.com/opencharly/spec/spec"
)

// A builder step exists because a candy needs the builder, so every missing piece of its context
// is an error — never the skip-comment that baked an image without the build (opencharly/sdk#608).
func TestEmitBuilderRefusesAMissingContext(t *testing.T) {
	view := spec.StepToView(&spec.BuilderStep{Builder: "npm", CandyName: "web"})
	env := spec.BuildEnv{Image: "box"}
	cases := map[string]*deploykit.Generator{
		"no image":      {Boxes: map[string]*buildkit.ResolvedBox{}},
		"no vocabulary": {Boxes: map[string]*buildkit.ResolvedBox{"box": {}}},
		"no candy": {Boxes: map[string]*buildkit.ResolvedBox{"box": {BuilderConfig: &spec.BuilderConfig{
			Builder: map[string]*spec.Builder{"npm": {DetectFiles: []string{"package.json"}}},
		}}}},
	}
	for name, dg := range cases {
		frag, err := emitBuilder(dg, view, env)
		if err == nil {
			t.Errorf("%s: emitBuilder returned %q and no error", name, frag)
			continue
		}
		if !strings.Contains(err.Error(), "npm") {
			t.Errorf("%s: error %q does not name the builder", name, err)
		}
	}
}

// A multi-stage builder's step never opens a stage: the image (box build or pod overlay) hoists the
// stage before its main FROM and copies the payload (D-IR-1, opencharly/sdk#608). The step renders
// only a declared package output its distro installs through local_pkg.
func TestEmitBuilderMultiStageRendersOnlyThePackageInstall(t *testing.T) {
	box := &buildkit.ResolvedBox{BuilderConfig: &spec.BuilderConfig{Builder: map[string]*spec.Builder{
		"npm": {DetectFiles: []string{"package.json"}},
		"aur": {},
	}}}
	dg := &deploykit.Generator{
		Boxes:   map[string]*buildkit.ResolvedBox{"box": box},
		Candies: map[string]deploykit.CandyModel{"web": deploykit.NewSpecCandyModel(spec.CandyModel{Name: "web"}, spec.CandyView{Name: "web"})},
	}
	env := spec.BuildEnv{Image: "box"}
	frag, err := emitBuilder(dg, spec.StepToView(&spec.BuilderStep{Builder: "npm", CandyName: "web"}), env)
	if err != nil || frag != "" {
		t.Fatalf("npm step fragment = %q, %v; want none (the overlay hoists the stage and copies the payload)", frag, err)
	}
	aur := &spec.BuilderStep{Builder: "aur", CandyName: "web",
		Package:  &spec.BuilderPackage{Format: "pac", Dir: "/tmp/aur-pkgs/web", Glob: "*.pkg.tar.zst"},
		LocalPkg: &spec.LocalPkg{InstallTemplate: "pacman -U --noconfirm {{.StageDir}}/{{.Glob}}\n", Probe: "command -v pacman"},
	}
	frag, err = emitBuilder(dg, spec.StepToView(aur), env)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"COPY --from=web-aur-build /tmp/aur-pkgs/web/ /tmp/aur-pkgs/web/", "pacman -U --noconfirm /tmp/aur-pkgs/web/*.pkg.tar.zst"} {
		if !strings.Contains(frag, want) {
			t.Errorf("aur step fragment missing %q:\n%s", want, frag)
		}
	}
	if strings.Contains(frag, "FROM ") {
		t.Errorf("a builder step opened a stage:\n%s", frag)
	}
}
