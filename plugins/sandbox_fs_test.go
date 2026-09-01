package plugins

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type sandboxInput struct {
	Operation  string `json:"operation"`
	MountPoint string `json:"mount_point,omitempty"`
	FilePath   string `json:"file_path,omitempty"`
	Content    string `json:"content,omitempty"`
	Target     string `json:"target,omitempty"`
}

type sandboxOutput struct {
	FileContent string  `json:"file_content,omitempty"`
	Error       *string `json:"error,omitempty"`
}

// startSandboxManager loads test-library against libraryDir with the given grant.
func startSandboxManager(tmpDir, libraryDir string, grant func(*model.Plugin)) *Manager {
	GinkgoHelper()
	installed := installTestPlugins(tmpDir, "test-library"+PackageExtension)
	grant(&installed[0])

	DeferCleanup(configtest.SetupConfig())
	conf.Server.Plugins.Enabled = true
	conf.Server.Plugins.Folder = conf.NewDir(tmpDir)
	conf.Server.Plugins.AutoReload = false

	mockPluginRepo := tests.CreateMockPluginRepo()
	mockPluginRepo.Permitted = true
	mockPluginRepo.SetData(installed)

	mockLibraryRepo := &tests.MockLibraryRepo{}
	mockLibraryRepo.SetData(model.Libraries{{ID: 1, Name: "Test Library", Path: libraryDir}})

	manager := &Manager{
		plugins:        make(map[string]*plugin),
		ds:             &tests.MockDataStore{MockedPlugin: mockPluginRepo, MockedLibrary: mockLibraryRepo},
		subsonicRouter: http.NotFoundHandler(),
	}
	Expect(manager.Start(GinkgoT().Context())).To(Succeed())
	DeferCleanup(func() { _ = manager.Stop() })
	return manager
}

// callSandbox runs one filesystem operation inside the plugin sandbox.
func callSandbox(manager *Manager, input sandboxInput) sandboxOutput {
	GinkgoHelper()
	manager.mu.RLock()
	p := manager.plugins["test-library"]
	manager.mu.RUnlock()
	Expect(p).ToNot(BeNil())

	instance, err := p.instance(context.Background())
	Expect(err).ToNot(HaveOccurred())
	defer instance.Close(context.Background())

	inputBytes, err := json.Marshal(input)
	Expect(err).ToNot(HaveOccurred())
	_, outputBytes, err := instance.Call("nd_test_library", inputBytes)
	Expect(err).ToNot(HaveOccurred())

	var output sandboxOutput
	Expect(json.Unmarshal(outputBytes, &output)).To(Succeed())
	return output
}

var _ = Describe("Plugin filesystem sandbox", Ordered, ContinueOnFailure, func() {
	var (
		manager    *Manager
		libraryDir string
		outsideDir string
		secretFile string
		mountPoint string
	)

	call := func(input sandboxInput) sandboxOutput {
		GinkgoHelper()
		input.MountPoint = mountPoint
		return callSandbox(manager, input)
	}

	BeforeAll(func() {
		tmpDir := GinkgoT().TempDir()
		libraryDir = filepath.Join(tmpDir, "music-library")
		Expect(os.MkdirAll(libraryDir, 0755)).To(Succeed())

		// Sibling of the mount root: anything reached here escaped the sandbox
		outsideDir = filepath.Join(tmpDir, "outside")
		Expect(os.MkdirAll(outsideDir, 0755)).To(Succeed())
		secretFile = filepath.Join(outsideDir, "secret.txt")
		Expect(os.WriteFile(secretFile, []byte("secret"), 0600)).To(Succeed())

		mountPoint = toPluginMountPoint(1)
		manager = startSandboxManager(tmpDir, libraryDir, func(p *model.Plugin) {
			p.AllLibraries = true
			p.AllowWriteAccess = true
		})
	})

	It("allows writing inside the mount", func() {
		out := call(sandboxInput{Operation: "write_file", FilePath: "inside.txt", Content: "hello"})

		Expect(out.Error).To(BeNil())
		Expect(os.ReadFile(filepath.Join(libraryDir, "inside.txt"))).To(BeEquivalentTo("hello"))
	})

	It("cannot escape the mount by creating a symlink", func() {
		linked := call(sandboxInput{Operation: "symlink", FilePath: "escape", Target: "../outside"})
		Expect(linked.Error).ToNot(BeNil())
		_, err := os.Lstat(filepath.Join(libraryDir, "escape"))
		Expect(err).To(MatchError(os.ErrNotExist), "the plugin created a symlink out of the mount")

		wrote := call(sandboxInput{Operation: "write_file", FilePath: "escape/via-created-symlink.txt", Content: "escaped"})

		Expect(wrote.Error).ToNot(BeNil())
		Expect(filepath.Join(outsideDir, "via-created-symlink.txt")).ToNot(BeAnExistingFile())
	})

	// Accepted residual, pinned so a future tightening can't happen silently
	It("still follows a symlink planted in the mount by something else", func() {
		Expect(os.Symlink(outsideDir, filepath.Join(libraryDir, "planted"))).To(Succeed())

		out := call(sandboxInput{Operation: "write_file", FilePath: "planted/via-symlink.txt", Content: "escaped"})

		Expect(out.Error).To(BeNil())
		Expect(filepath.Join(outsideDir, "via-symlink.txt")).To(BeAnExistingFile())
	})

	It("cannot read outside the mount with ..", func() {
		out := call(sandboxInput{Operation: "read_file", FilePath: "../outside/secret.txt"})

		Expect(out.Error).ToNot(BeNil())
		Expect(out.FileContent).ToNot(Equal("secret"))
	})

	It("cannot write outside the mount with ..", func() {
		out := call(sandboxInput{Operation: "write_file", FilePath: "../outside/via-dotdot.txt", Content: "escaped"})

		Expect(out.Error).ToNot(BeNil())
		Expect(filepath.Join(outsideDir, "via-dotdot.txt")).ToNot(BeAnExistingFile())
	})
})

var _ = Describe("Plugin filesystem sandbox without library access", Ordered, func() {
	var manager *Manager
	var libraryDir string

	BeforeAll(func() {
		tmpDir := GinkgoT().TempDir()
		libraryDir = filepath.Join(tmpDir, "music-library")
		Expect(os.MkdirAll(libraryDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(libraryDir, "track.txt"), []byte("audio"), 0600)).To(Succeed())

		manager = startSandboxManager(tmpDir, libraryDir, func(*model.Plugin) {})
	})

	It("mounts nothing when no library is granted", func() {
		manager.mu.RLock()
		p := manager.plugins["test-library"]
		manager.mu.RUnlock()
		Expect(p.fsConfig).To(BeNil())

		out := callSandbox(manager, sandboxInput{Operation: "read_file", MountPoint: toPluginMountPoint(1), FilePath: "track.txt"})
		Expect(out.Error).ToNot(BeNil())
	})
})
