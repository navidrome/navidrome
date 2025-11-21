package mpv

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MPV", func() {
	var (
		testScript string
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

		// Create mock MPV script that outputs arguments to stdout
		testScript = createMockMPVScript(tempDir)

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
					"/tmp/mpv.sh",
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
		Context("with paths containing spaces in template arguments", func() {
			BeforeEach(func() {
				// Template with spaces in the path arguments themselves
				conf.Server.MPVCmdTemplate = `mpv --no-audio-display --pause %f --ao-pcm-file="/audio/my folder/snapcast_fifo" --input-ipc-server=%s`
			})

			It("handles spaces in quoted template argument paths", func() {
				args := createMPVCommand("auto", "/music/test.mp3", "/tmp/socket")
				// This test reveals the limitation of strings.Fields() - it will split on all spaces
				// Expected behavior would be to keep the path as one argument
				Expect(args).To(Equal([]string{
					testScript,
					"--no-audio-display",
					"--pause",
					"/music/test.mp3",
					"--ao-pcm-file=/audio/my folder/snapcast_fifo", // This should be one argument
					"--input-ipc-server=/tmp/socket",
				}))
			})
		})

		Context("with malformed template", func() {
			BeforeEach(func() {
				// Template with unmatched quotes that will cause shell parsing to fail
				conf.Server.MPVCmdTemplate = `mpv --no-audio-display --pause %f --input-ipc-server=%s --ao-pcm-file="/unclosed/quote`
			})

			It("returns nil when shell parsing fails", func() {
				args := createMPVCommand("auto", "/music/test.mp3", "/tmp/socket")
				Expect(args).To(BeNil())
			})
		})

		Context("with empty template", func() {
			BeforeEach(func() {
				conf.Server.MPVCmdTemplate = ""
			})

			It("returns empty slice for empty template", func() {
				args := createMPVCommand("auto", "/music/test.mp3", "/tmp/socket")
				Expect(args).To(Equal([]string{}))
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

			// Read all the output from stdout (this will block until the process finishes or is canceled)
			output, err := io.ReadAll(executor)
			Expect(err).ToNot(HaveOccurred())

			// Parse the captured arguments
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
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

			// Read all the output from stdout (this will block until the process finishes or is canceled)
			output, err := io.ReadAll(executor)
			Expect(err).ToNot(HaveOccurred())

			// Parse the captured arguments
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
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

				// Read all the output from stdout (this will block until the process finishes or is canceled)
				output, err := io.ReadAll(executor)
				Expect(err).ToNot(HaveOccurred())

				// Parse the captured arguments
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")

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

		Context("with nil args", func() {
			It("returns error when args is nil", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				_, err := start(ctx, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("no command arguments provided"))
			})

			It("returns error when args is empty", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				_, err := start(ctx, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("no command arguments provided"))
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

	Describe("NewTrack integration", func() {
		var testMediaFile model.MediaFile

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.MPVPath = testScript

			// Create a test media file
			testMediaFile = model.MediaFile{
				ID:   "test-id",
				Path: "/music/test.mp3",
			}
		})

		Context("with malformed template", func() {
			BeforeEach(func() {
				// Template with unmatched quotes that will cause shell parsing to fail
				conf.Server.MPVCmdTemplate = `mpv --no-audio-display --pause %f --input-ipc-server=%s --ao-pcm-file="/unclosed/quote`
			})

			It("returns error when createMPVCommand fails", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				playbackDone := make(chan bool, 1)
				_, err := NewTrack(ctx, playbackDone, "auto", testMediaFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("no mpv command arguments provided"))
			})
		})
	})
})

// createMockMPVScript creates a mock script that outputs arguments to stdout
func createMockMPVScript(tempDir string) string {
	var scriptContent string
	var scriptExt string

	if runtime.GOOS == "windows" {
		scriptExt = ".bat"
		scriptContent = `@echo off
echo %0
:loop
if "%~1"=="" goto end
echo %~1
shift
goto loop
:end
`
	} else {
		scriptExt = ".sh"
		scriptContent = `#!/bin/sh
echo "$0"
for arg in "$@"; do
    echo "$arg"
done
`
	}

	scriptPath := filepath.Join(tempDir, "mock_mpv"+scriptExt)
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755) // nolint:gosec
	if err != nil {
		panic(fmt.Sprintf("Failed to create mock script: %v", err))
	}

	return scriptPath
}
