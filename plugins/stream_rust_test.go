package plugins

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// buildRustTestPlugin skips when the wasm32-wasip1 Rust toolchain is missing; CI has no Rust.
func buildRustTestPlugin(dir string) string {
	rustup, err := exec.LookPath("rustup")
	if err != nil {
		Skip("rustup is not installed")
	}
	targets, err := exec.Command(rustup, "target", "list", "--installed").Output()
	if err != nil || !strings.Contains(string(targets), "wasm32-wasip1") {
		Skip("the wasm32-wasip1 Rust target is not installed")
	}
	cargo, err := exec.Command(rustup, "which", "cargo").Output()
	Expect(err).ToNot(HaveOccurred())
	rustc, err := exec.Command(rustup, "which", "rustc").Output()
	Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command(strings.TrimSpace(string(cargo)), "build", "--release", "--target", "wasm32-wasip1") //nolint:gosec // path comes from rustup
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CARGO_BUILD_RUSTC="+strings.TrimSpace(string(rustc)))
	out, err := cmd.CombinedOutput()
	Expect(err).ToNot(HaveOccurred(), string(out))

	name := filepath.Base(dir)
	wasm := filepath.Join(dir, "target", "wasm32-wasip1", "release", strings.ReplaceAll(name, "-", "_")+".wasm")
	pkg := name + PackageExtension
	Expect(packageFiles(filepath.Join(testdataDir, pkg), filepath.Join(dir, manifestFileName), wasm)).To(Succeed())
	return pkg
}

var _ = Describe("Rust plugin streams", Ordered, func() {
	var (
		ctx context.Context
		p   *plugin
	)

	BeforeAll(func() {
		pkg := buildRustTestPlugin(filepath.Join(testdataDir, "test-streams-rs"))
		manager, _ := createTestManagerWithPlugins(nil, pkg)
		p = loadedTestPlugin(manager, "test-streams-rs")
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("reads a host stream with std::io::Read", func() {
		out, err := callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, "read_all", defaultTimeout,
			readInput(io.NopCloser(strings.NewReader("streamed to rust")), 0), nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(out.Data).To(Equal("streamed to rust"))
	})

	It("reads into a slice that starts at an odd address", func() {
		out, err := callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, "read_unaligned", defaultTimeout,
			readInput(io.NopCloser(strings.NewReader("misaligned bytes here")), 0), nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(out.Data).To(Equal("misaligned bytes here"))
	})

	It("writes from a slice that starts at an odd address", func() {
		w := &trackingWriter{}
		out, err := callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, "write_unaligned", defaultTimeout,
			func(s *callScope) (streamsInput, error) {
				h, err := s.register(newWriteStream(w))
				return streamsInput{Handle: h}, err
			}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(string(w.bytes())).To(Equal("unaligned write, 8+ bytes"))
	})

	It("writes a host stream with std::io::Write and closes it", func() {
		w := &trackingWriter{}
		out, err := callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, "write_greeting", defaultTimeout,
			func(s *callScope) (streamsInput, error) {
				h, err := s.register(newWriteStream(w))
				return streamsInput{Handle: h}, err
			}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(string(w.bytes())).To(Equal("hello from rust"))
		Expect(w.closed()).To(BeTrue())
	})
})
