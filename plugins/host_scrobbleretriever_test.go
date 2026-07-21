//go:build !windows

package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scrobbble Retriever Host Function", Ordered, func() {
	var (
		manager   *Manager
		tmpDir    string
		dataStore *tests.MockDataStore
	)

	BeforeAll(func() {
		ctx := GinkgoT().Context()

		var err error
		tmpDir, err = os.MkdirTemp("", "scrobble-retriever-test-*")
		Expect(err).ToNot(HaveOccurred())

		conf.Server.DbPath = filepath.Join(tmpDir, "test-scanner.db?_journal_mode=WAL")

		db.Init(ctx)
		DeferCleanup(func() {
			Expect(tests.ClearDB()).To(Succeed())
		})
		dataStore = &tests.MockDataStore{RealDS: persistence.New(db.Db())}

		// Copy test plugin to temp dir
		srcPath := filepath.Join(testdataDir, "test-scrobble-retriever"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-scrobble-retriever"+PackageExtension)
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = conf.NewDir(tmpDir)
		conf.Server.Plugins.AutoReload = false

		userRepo := dataStore.User(ctx)
		// Add test users
		_ = userRepo.Put(&model.User{
			ID:       "user1",
			UserName: "testuser",
			IsAdmin:  false,
		})
		_ = userRepo.Put(&model.User{
			ID:       "admin1",
			UserName: "adminuser",
			IsAdmin:  true,
		})

		err = dataStore.MediaFile(ctx).Put(&model.MediaFile{ID: "1", LibraryID: 1})
		Expect(err).To(BeNil())
		err = dataStore.MediaFile(ctx).Put(&model.MediaFile{ID: "2", LibraryID: 1})
		Expect(err).To(BeNil())
		err = dataStore.MediaFile(ctx).Put(&model.MediaFile{ID: "3", LibraryID: 1})
		Expect(err).To(BeNil())

		scrobbleCtx := request.WithUser(GinkgoT().Context(), model.User{ID: "admin1", UserName: "adminuser"})

		scrobbleRepo := dataStore.Scrobble(scrobbleCtx)
		err = scrobbleRepo.RecordScrobble("1", time.Unix(0, 0))
		Expect(err).To(BeNil())
		err = scrobbleRepo.RecordScrobble("2", time.Unix(1, 0))
		Expect(err).To(BeNil())
		err = scrobbleRepo.RecordScrobble("3", time.Unix(2, 0))
		Expect(err).To(BeNil())
		err = scrobbleRepo.RecordScrobble("1", time.Unix(2, 0))
		Expect(err).To(BeNil())

		// Create and configure manager
		manager = &Manager{
			plugins: make(map[string]*plugin),
			ds:      dataStore,
		}
		router := &fakeSubsonicRouter{}
		manager.SetSubsonicRouter(router)

		// Pre-enable the plugin in the mock repo so it loads on startup
		// Compute SHA256 of the plugin file to match what syncPlugins will compute
		pluginPath := filepath.Join(tmpDir, "test-scrobble-retriever"+PackageExtension)
		wasmData, err := os.ReadFile(pluginPath)
		Expect(err).ToNot(HaveOccurred())
		hash := sha256.Sum256(wasmData)
		hashHex := hex.EncodeToString(hash[:])

		dataStore.MockedPlugin = tests.CreateMockPluginRepo()

		mockPluginRepo := dataStore.Plugin(GinkgoT().Context()).(*tests.MockPluginRepo)
		mockPluginRepo.Permitted = true
		enabledPlugin := model.Plugin{
			ID:      "test-scrobble-retriever",
			Path:    pluginPath,
			SHA256:  hashHex,
			Enabled: true,
			Users:   `["user1","admin1"]`,
		}
		mockPluginRepo.SetData(model.Plugins{enabledPlugin})

		// Start the manager
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	Describe("no items", func() {
		var plugin *plugin

		BeforeEach(func() {
			manager.mu.RLock()
			plugin = manager.plugins["test-scrobble-retriever"]
			manager.mu.RUnlock()
			Expect(plugin).ToNot(BeNil())
		})

		It("calls get first timestamp", func() {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, output, err := instance.Call("call_get_first_timestamp", []byte("testuser"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))

			Expect(output).To(Equal([]byte("{\"timestamp\":null}")))
		})

		It("calls get last timestamp", func() {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, output, err := instance.Call("call_get_last_timestamp", []byte("testuser"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))

			Expect(output).To(Equal([]byte("{\"timestamp\":null}")))
		})

		It("calls scrobbles", func() {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, output, err := instance.Call("call_get_scrobbles", []byte(`{"username":"testuser"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))

			Expect(output).To(Equal([]byte(`{"scrobbles":[],"nextTimestamp":null}`)))
		})

		It("calls get scrobble count", func() {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, output, err := instance.Call("call_get_scrobbles_count", []byte(`{"username":"testuser"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))
			Expect(output).To(Equal([]byte("0")))
		})
	})

	Describe("with items", func() {
		var plugin *plugin

		p := func(val int64) *int64 {
			return &val
		}

		scrobbles := []host.ScrobbleRef{
			{ID: 1, MediaFileID: "1", SubmissionTime: 0},
			{ID: 2, MediaFileID: "2", SubmissionTime: 1},
			{ID: 3, MediaFileID: "3", SubmissionTime: 2},
			{ID: 4, MediaFileID: "1", SubmissionTime: 2},
		}

		scrobblesReversed := make([]host.ScrobbleRef, 4)

		BeforeAll(func() {
			for idx := range scrobbles {
				scrobblesReversed[3-idx] = scrobbles[idx]
			}
		})

		BeforeEach(func() {
			manager.mu.RLock()
			plugin = manager.plugins["test-scrobble-retriever"]
			manager.mu.RUnlock()
			Expect(plugin).ToNot(BeNil())
		})

		It("calls get first timestamp", func() {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, output, err := instance.Call("call_get_first_timestamp", []byte("adminuser"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))

			Expect(output).To(Equal([]byte("{\"timestamp\":0}")))
		})

		It("calls get last timestamp", func() {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, output, err := instance.Call("call_get_last_timestamp", []byte("adminuser"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))

			Expect(output).To(Equal([]byte("{\"timestamp\":2}")))
		})

		DescribeTable("getScrobbles", func(params string, scrobbles []host.ScrobbleRef, timestamp *int64) {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, output, err := instance.Call("call_get_scrobbles", []byte(params))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))

			var scrobbleList host.ScrobbleList
			Expect(json.Unmarshal(output, &scrobbleList)).To(Succeed())
			Expect(scrobbleList).To(Equal(host.ScrobbleList{
				Scrobbles:     scrobbles,
				NextTimestamp: timestamp,
			}))
		},
			Entry("calls scrobbles in ascending order", `{"username":"adminuser"}`, scrobbles, nil),
			Entry("calls scrobbles in ascending order, beyond range", `{"username":"adminuser","fromTimestamp":-1, "toTimestamp": 1000}`, scrobbles, nil),
			Entry("calls subset of scrobbles in ascending order, next timestamp", `{"username":"adminuser","maxItems":2}`, scrobbles[:2], p(2)),
			Entry("calls subset of scrobbles in ascending order, with offset next timestamp", `{"username":"adminuser","maxItems":2,"fromTimestamp":1}`, scrobbles[1:3], p(2)),
			Entry("calls subset of scrobbles in ascending order, from and to timestamp", `{"username":"adminuser","toTimestamp":2,"fromTimestamp":1}`, scrobbles[1:], nil),
			Entry("calls in reverse order, full", `{"username":"adminuser","toTimestamp":2}`, scrobblesReversed, nil),
			Entry("calls in reverse order, with count", `{"username":"adminuser","toTimestamp":2, "maxItems": 3}`, scrobblesReversed[:3], p(0)),
			Entry("calls in reverse order, with count of 1", `{"username":"adminuser","toTimestamp":2, "maxItems": 1}`, scrobblesReversed[:1], p(2)),
		)

		DescribeTable("GetScrobblesCount", func(params string, count int) {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, output, err := instance.Call("call_get_scrobbles_count", []byte(params))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))

			value, err := strconv.ParseInt(string(output), 10, 64)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(int64(count)))
		},
			Entry("gets all scrobbles", `{"username":"adminuser"}`, 4),
			Entry("gets two scrobbles ascending", `{"username":"adminuser", "fromTimestamp": 2}`, 2),
			Entry("gets one scrobble descending", `{"username":"adminuser", "toTimestamp": 0}`, 1),
			Entry("filters upper and bottom", `{"username":"adminuser", "fromTimestamp": 1, "toTimestamp": 1}`, 1),
			Entry("accepts filter out of range", `{"username":"adminuser", "fromTimestamp": -1, "toTimestamp": 1000}`, 4),
		)
	})
})
