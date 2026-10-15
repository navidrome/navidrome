package plugins

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// buildTestPlugins packages every test plugin under dir, replicating
// `make -C plugins/testdata` without needing make or zip on the PATH.
func buildTestPlugins(dir string) {
	start := time.Now()
	built, err := buildPackages(dir)
	fmt.Fprintf(GinkgoWriter, "[BeforeSuite] built test plugins in %s: %v\n", time.Since(start), built)
	Expect(err).ToNot(HaveOccurred(), "failed to build test plugins")
}

func buildPackages(dir string) ([]string, error) {
	mods, err := filepath.Glob(filepath.Join(dir, "*", "go.mod"))
	if err != nil || len(mods) == 0 {
		return nil, err
	}
	pdkTime, err := newestModTime(filepath.Join(dir, "..", "pdk", "go"))
	if err != nil {
		return nil, err
	}

	built := make([]string, len(mods))
	errs := make([]error, len(mods))
	build := func(i int) {
		pluginDir := filepath.Dir(mods[i])
		var rebuilt bool
		if rebuilt, errs[i] = buildPackage(pluginDir, pdkTime); rebuilt {
			built[i] = filepath.Base(pluginDir)
		}
	}

	// The first build populates the wasip1 stdlib and PDK objects every plugin
	// shares; fanning out before it lands makes each one compile them again.
	build(0)
	var wg sync.WaitGroup
	for i := range mods[1:] {
		wg.Go(func() { build(i + 1) })
	}
	wg.Wait()
	return slices.DeleteFunc(built, func(name string) bool { return name == "" }), errors.Join(errs...)
}

func buildPackage(dir string, pdkTime time.Time) (bool, error) {
	sourceTime, err := newestModTime(dir)
	if err != nil {
		return false, err
	}
	pkg := dir + PackageExtension
	if info, err := os.Stat(pkg); err == nil && info.ModTime().After(utils.TimeNewest(sourceTime, pdkTime)) {
		return false, nil
	}

	wasm, err := filepath.Abs(pkg + ".build.wasm")
	if err != nil {
		return false, err
	}
	defer os.Remove(wasm)
	// -buildvcs=false keeps the bytes stable across commits, so the suite's
	// wazero compilation cache still hits after a rebuild.
	cmd := exec.Command("go", "build", "-buildvcs=false", "-buildmode=c-shared", "-o", wasm, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("building %s: %w\n%s", dir, err, out)
	}

	tmp := pkg + ".build.ndp"
	defer os.Remove(tmp)
	if err := packageFiles(tmp, filepath.Join(dir, manifestFileName), wasm); err != nil {
		return false, fmt.Errorf("packaging %s: %w", dir, err)
	}
	return true, os.Rename(tmp, pkg)
}

func packageFiles(pkg, manifest, wasm string) error {
	m, err := os.Open(manifest)
	if err != nil {
		return err
	}
	defer m.Close()
	w, err := os.Open(wasm)
	if err != nil {
		return err
	}
	defer w.Close()
	return writeNdp(pkg, m, w)
}

func newestModTime(root string) (time.Time, error) {
	var newest time.Time
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		newest = utils.TimeNewest(newest, info.ModTime())
		return nil
	})
	return newest, err
}
