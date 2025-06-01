package mpv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMPV(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "MPV Suite")
}

var _ = Describe("MPV", func() {
	var (
		testScript string
		outputFile string
		tempDir    string
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())

		// Reset MPV cache
		mpvOnce = sync.Once{}
		mpvPath = ""
		mpvErr = nil

		// Create temporary directory for test files
		var err error
		tempDir, err = os.MkdirTemp("", "mpv_test_*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { os.RemoveAll(tempDir) })

		// Create output file to capture command arguments
		outputFile = filepath.Join(tempDir, "mpv_args.txt")

		// Create mock MPV script that captures arguments
		testScript = createMockMPVScript(tempDir, outputFile)

		// Configure test MPV path
		conf.Server.MPVPath = testScript
	})

	Describe("createMPVCommand", func() {
		Context("with default template", func() {
			BeforeEach(func() {
				conf.Server.MPVCmdTemplate = "mpv --audio-device=%d --no-audio-display --pause %f --input-ipc-server=%s"
			})

			It("creates correct command with simple paths", func() {
				args := createMPVCommand("auto", "/music/test.mp3", "/tmp/socket")
				Expect(args).To(Equal([]string{
					testScript,
					"--audio-device=auto",
					"--no-audio-display",
					"--pause",
					"/music/test.mp3",
					"--input-ipc-server=/tmp/socket",
				}))
			})

			It("handles paths with spaces", func() {
				args := createMPVCommand("auto", "/music/My Album/01 - Song.mp3", "/tmp/socket")
				Expect(args).To(Equal([]string{
					testScript,
					"--audio-device=auto",
					"--no-audio-display",
					"--pause",
					"/music/My Album/01 - Song.mp3",
					"--input-ipc-server=/tmp/socket",
				}))
			})

			It("handles complex device names", func() {
				deviceName := "coreaudio/AppleUSBAudioEngine:Cambridge Audio :Cambridge Audio USB Audio 1.0:0000:1"
				args := createMPVCommand(deviceName, "/music/test.mp3", "/tmp/socket")
				Expect(args).To(Equal([]string{
					testScript,
					"--audio-device=" + deviceName,
					"--no-audio-display",
					"--pause",
					"/music/test.mp3",
					"--input-ipc-server=/tmp/socket",
				}))
			})
		})

		Context("with snapcast template (issue #3619)", func() {
			BeforeEach(func() {
				// This is the template that fails with naive space splitting
				conf.Server.MPVCmdTemplate = "mpv --no-audio-display --pause %f --input-ipc-server=%s --audio-channels=stereo --audio-samplerate=48000 --audio-format=s16 --ao=pcm --ao-pcm-file=/audio/snapcast_fifo"
			})

			It("creates correct command for snapcast integration", func() {
				args := createMPVCommand("auto", "/music/test.mp3", "/tmp/socket")
				Expect(args).To(Equal([]string{
					testScript,
					"--no-audio-display",
					"--pause",
					"/music/test.mp3",
					"--input-ipc-server=/tmp/socket",
					"--audio-channels=stereo",
					"--audio-samplerate=48000",
					"--audio-format=s16",
					"--ao=pcm",
					"--ao-pcm-file=/audio/snapcast_fifo",
				}))
			})
		})

		Context("with wrapper script template", func() {
			BeforeEach(func() {
				// Test case that would break with naive splitting due to quoted arguments
				conf.Server.MPVCmdTemplate = `/tmp/mpv.sh --no-audio-display --pause %f --input-ipc-server=%s --audio-channels=stereo`
			})

			It("handles wrapper script paths", func() {
				args := createMPVCommand("auto", "/music/test.mp3", "/tmp/socket")
				Expect(args).To(Equal([]string{
					testScript, // This will be substituted by fixCmd
					"--no-audio-display",
					"--pause",
					"/music/test.mp3",
					"--input-ipc-server=/tmp/socket",
					"--audio-channels=stereo",
				}))
			})
		})

		Context("with extra spaces in template", func() {
			BeforeEach(func() {
				conf.Server.MPVCmdTemplate = "mpv    --audio-device=%d   --no-audio-display     --pause %f --input-ipc-server=%s"
			})

			It("handles extra spaces correctly", func() {
				args := createMPVCommand("auto", "/music/test.mp3", "/tmp/socket")
				Expect(args).To(Equal([]string{
					testScript,
					"--audio-device=auto",
					"--no-audio-display",
					"--pause",
					"/music/test.mp3",
					"--input-ipc-server=/tmp/socket",
				}))
			})
		})
	})

	Describe("start", func() {
		BeforeEach(func() {
			conf.Server.MPVCmdTemplate = "mpv --audio-device=%d --no-audio-display --pause %f --input-ipc-server=%s"
		})

		It("executes MPV command and captures arguments correctly", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			deviceName := "auto"
			filename := "/music/test.mp3"
			socketName := "/tmp/test_socket"

			args := createMPVCommand(deviceName, filename, socketName)
			executor, err := start(ctx, args)
			Expect(err).ToNot(HaveOccurred())

			// Wait a bit for the mock script to write the output
			time.Sleep(100 * time.Millisecond)

			// Cancel the executor
			err = executor.Cancel()
			Expect(err).ToNot(HaveOccurred())

			// Read and verify the captured arguments
			Eventually(func() bool {
				_, err := os.Stat(outputFile)
				return err == nil
			}, "2s", "100ms").Should(BeTrue(), "Mock script should have created output file")

			content, err := os.ReadFile(outputFile)
			Expect(err).ToNot(HaveOccurred())

			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			Expect(lines).To(HaveLen(6))
			Expect(lines[0]).To(Equal(testScript))
			Expect(lines[1]).To(Equal("--audio-device=auto"))
			Expect(lines[2]).To(Equal("--no-audio-display"))
			Expect(lines[3]).To(Equal("--pause"))
			Expect(lines[4]).To(Equal("/music/test.mp3"))
			Expect(lines[5]).To(Equal("--input-ipc-server=/tmp/test_socket"))
		})

		It("handles file paths with spaces", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			deviceName := "auto"
			filename := "/music/My Album/01 - My Song.mp3"
			socketName := "/tmp/test socket"

			args := createMPVCommand(deviceName, filename, socketName)
			executor, err := start(ctx, args)
			Expect(err).ToNot(HaveOccurred())

			// Wait a bit for the mock script to write the output
			time.Sleep(100 * time.Millisecond)

			// Cancel the executor
			err = executor.Cancel()
			Expect(err).ToNot(HaveOccurred())

			// Read and verify the captured arguments
			Eventually(func() bool {
				_, err := os.Stat(outputFile)
				return err == nil
			}, "2s", "100ms").Should(BeTrue())

			content, err := os.ReadFile(outputFile)
			Expect(err).ToNot(HaveOccurred())

			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			Expect(lines).To(ContainElement("/music/My Album/01 - My Song.mp3"))
			Expect(lines).To(ContainElement("--input-ipc-server=/tmp/test socket"))
		})

		Context("with complex snapcast configuration", func() {
			BeforeEach(func() {
				conf.Server.MPVCmdTemplate = "mpv --no-audio-display --pause %f --input-ipc-server=%s --audio-channels=stereo --audio-samplerate=48000 --audio-format=s16 --ao=pcm --ao-pcm-file=/audio/snapcast_fifo"
			})

			It("passes all snapcast arguments correctly", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				deviceName := "auto"
				filename := "/music/album/track.flac"
				socketName := "/tmp/mpv-ctrl-test.socket"

				args := createMPVCommand(deviceName, filename, socketName)
				executor, err := start(ctx, args)
				Expect(err).ToNot(HaveOccurred())

				// Wait a bit for the mock script to write the output
				time.Sleep(100 * time.Millisecond)

				// Cancel the executor
				err = executor.Cancel()
				Expect(err).ToNot(HaveOccurred())

				// Read and verify all arguments are passed correctly
				Eventually(func() bool {
					_, err := os.Stat(outputFile)
					return err == nil
				}, "2s", "100ms").Should(BeTrue())

				content, err := os.ReadFile(outputFile)
				Expect(err).ToNot(HaveOccurred())

				lines := strings.Split(strings.TrimSpace(string(content)), "\n")

				// Verify all expected arguments are present
				Expect(lines).To(ContainElement("--no-audio-display"))
				Expect(lines).To(ContainElement("--pause"))
				Expect(lines).To(ContainElement("/music/album/track.flac"))
				Expect(lines).To(ContainElement("--input-ipc-server=/tmp/mpv-ctrl-test.socket"))
				Expect(lines).To(ContainElement("--audio-channels=stereo"))
				Expect(lines).To(ContainElement("--audio-samplerate=48000"))
				Expect(lines).To(ContainElement("--audio-format=s16"))
				Expect(lines).To(ContainElement("--ao=pcm"))
				Expect(lines).To(ContainElement("--ao-pcm-file=/audio/snapcast_fifo"))
			})
		})
	})

	Describe("mpvCommand", func() {
		BeforeEach(func() {
			// Reset the mpv command cache
			mpvOnce = sync.Once{}
			mpvPath = ""
			mpvErr = nil
		})

		It("finds the configured MPV path", func() {
			conf.Server.MPVPath = testScript
			path, err := mpvCommand()
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(testScript))
		})
	})
})

// createMockMPVScript creates a mock script that captures the arguments passed to it
func createMockMPVScript(tempDir, outputFile string) string {
	var scriptContent string
	var scriptExt string

	if runtime.GOOS == "windows" {
		scriptExt = ".bat"
		scriptContent = fmt.Sprintf(`@echo off
echo %%0 > "%s"
:loop
if "%%~1"=="" goto end
echo %%~1 >> "%s"
shift
goto loop
:end
timeout /t 1 >nul
`, outputFile, outputFile)
	} else {
		scriptExt = ".sh"
		scriptContent = fmt.Sprintf(`#!/bin/bash
echo "$0" > "%s"
for arg in "$@"; do
    echo "$arg" >> "%s"
done
sleep 1
`, outputFile, outputFile)
	}

	scriptPath := filepath.Join(tempDir, "mock_mpv"+scriptExt)
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755) // nolint:gosec
	if err != nil {
		panic(fmt.Sprintf("Failed to create mock script: %v", err))
	}

	return scriptPath
}
