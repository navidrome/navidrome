package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AbsoluteURL", func() {
	When("BaseURL is empty", func() {
		BeforeEach(func() {
			conf.Server.BasePath = ""
		})
		It("uses the scheme/host from the request", func() {
			r, _ := http.NewRequest("GET", "https://myserver.com/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("https://myserver.com/share/img/123?a=xyz"))
		})
		It("does not override provided schema/host", func() {
			r, _ := http.NewRequest("GET", "http://localhost/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "http://public.myserver.com/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("http://public.myserver.com/share/img/123?a=xyz"))
		})
	})
	When("BaseURL has only path", func() {
		BeforeEach(func() {
			conf.Server.BasePath = "/music"
		})
		It("uses the scheme/host from the request", func() {
			r, _ := http.NewRequest("GET", "https://myserver.com/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("https://myserver.com/music/share/img/123?a=xyz"))
		})
		It("does not override provided schema/host", func() {
			r, _ := http.NewRequest("GET", "http://localhost/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "http://public.myserver.com/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("http://public.myserver.com/share/img/123?a=xyz"))
		})
	})
	When("BaseURL has full URL", func() {
		BeforeEach(func() {
			conf.Server.BaseScheme = "https"
			conf.Server.BaseHost = "myserver.com:8080"
			conf.Server.BasePath = "/music"
		})
		It("use the configured scheme/host/path", func() {
			r, _ := http.NewRequest("GET", "https://localhost:4533/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("https://myserver.com:8080/music/share/img/123?a=xyz"))
		})
		It("does not override provided schema/host", func() {
			r, _ := http.NewRequest("GET", "http://localhost/rest/ping?id=123", nil)
			actual := AbsoluteURL(r, "http://public.myserver.com/share/img/123", url.Values{"a": []string{"xyz"}})
			Expect(actual).To(Equal("http://public.myserver.com/share/img/123?a=xyz"))
		})
	})
})

var _ = Describe("createUnixSocketFile", func() {
	var socketPath string

	BeforeEach(func() {
		tempDir, _ := os.MkdirTemp("", "create_unix_socket_file_test")
		socketPath = filepath.Join(tempDir, "test.sock")
		DeferCleanup(func() {
			_ = os.RemoveAll(tempDir)
		})
	})

	When("unixSocketPerm is valid", func() {
		It("updates the permission of the unix socket file and returns nil", func() {
			_, err := createUnixSocketFile(socketPath, "0777")
			fileInfo, _ := os.Stat(socketPath)
			actualPermission := fileInfo.Mode().Perm()

			Expect(actualPermission).To(Equal(os.FileMode(0777)))
			Expect(err).ToNot(HaveOccurred())

		})
	})

	When("unixSocketPerm is invalid", func() {
		It("returns an error", func() {
			_, err := createUnixSocketFile(socketPath, "invalid")
			Expect(err).To(HaveOccurred())

		})
	})

	When("file already exists", func() {
		It("recreates the file as a socket with the right permissions", func() {
			_, err := os.Create(socketPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.Chmod(socketPath, os.FileMode(0777))).To(Succeed())

			_, err = createUnixSocketFile(socketPath, "0600")
			Expect(err).ToNot(HaveOccurred())
			fileInfo, _ := os.Stat(socketPath)
			Expect(fileInfo.Mode().Perm()).To(Equal(os.FileMode(0600)))
			Expect(fileInfo.Mode().Type()).To(Equal(fs.ModeSocket))
		})
	})
})

var _ = Describe("TLS support", func() {
	Describe("validateTLSCertificates", func() {
		const testDataDir = "server/testdata"

		When("certificate and key are valid and unencrypted", func() {
			It("returns nil", func() {
				certFile := filepath.Join(testDataDir, "test_cert.pem")
				keyFile := filepath.Join(testDataDir, "test_key.pem")
				err := validateTLSCertificates(certFile, keyFile)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("private key is encrypted with PKCS#8 format", func() {
			It("returns an error with helpful message", func() {
				certFile := filepath.Join(testDataDir, "test_cert_encrypted.pem")
				keyFile := filepath.Join(testDataDir, "test_key_encrypted.pem")
				err := validateTLSCertificates(certFile, keyFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("encrypted"))
				Expect(err.Error()).To(ContainSubstring("openssl"))
			})
		})

		When("private key is encrypted with legacy format (Proc-Type header)", func() {
			It("returns an error with helpful message", func() {
				certFile := filepath.Join(testDataDir, "test_cert.pem")
				keyFile := filepath.Join(testDataDir, "test_key_encrypted_legacy.pem")
				err := validateTLSCertificates(certFile, keyFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("encrypted"))
				Expect(err.Error()).To(ContainSubstring("openssl"))
			})
		})

		When("key file does not exist", func() {
			It("returns an error", func() {
				certFile := filepath.Join(testDataDir, "test_cert.pem")
				keyFile := filepath.Join(testDataDir, "nonexistent.pem")
				err := validateTLSCertificates(certFile, keyFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("reading TLS key file"))
			})
		})

		When("key file does not contain valid PEM", func() {
			It("returns an error", func() {
				// Create a temp file with invalid PEM content
				tmpFile, err := os.CreateTemp("", "invalid_key*.pem")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					_ = os.Remove(tmpFile.Name())
				})
				_, err = tmpFile.WriteString("not a valid PEM file")
				Expect(err).ToNot(HaveOccurred())
				_ = tmpFile.Close()

				certFile := filepath.Join(testDataDir, "test_cert.pem")
				err = validateTLSCertificates(certFile, tmpFile.Name())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("valid PEM block"))
			})
		})

		When("certificate file does not exist", func() {
			It("returns an error from tls.LoadX509KeyPair", func() {
				certFile := filepath.Join(testDataDir, "nonexistent_cert.pem")
				keyFile := filepath.Join(testDataDir, "test_key.pem")
				err := validateTLSCertificates(certFile, keyFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("loading TLS certificate/key pair"))
			})
		})
	})

	Describe("Server TLS", func() {
		const testDataDir = "server/testdata"

		When("server is started with valid TLS certificates", func() {
			It("accepts HTTPS connections", func() {
				DeferCleanup(configtest.SetupConfig())

				// Create server with mock dependencies
				ds := &tests.MockDataStore{}
				server := New(ds, nil, nil)

				// Load the test certificate to create a trusted CA pool
				certFile := filepath.Join(testDataDir, "test_cert.pem")
				keyFile := filepath.Join(testDataDir, "test_key.pem")
				caCert, err := os.ReadFile(certFile)
				Expect(err).ToNot(HaveOccurred())

				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)

				// Create an HTTPS client that trusts our test certificate
				httpClient := &http.Client{
					Timeout: 5 * time.Second,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							RootCAs:    caCertPool,
							MinVersion: tls.VersionTLS12,
						},
					},
				}

				// Start the server in a goroutine
				ctx, cancel := context.WithCancel(GinkgoT().Context())
				defer cancel()

				errChan := make(chan error, 1)
				go func() {
					errChan <- server.Run(ctx, "127.0.0.1", 14534, certFile, keyFile)
				}()

				Eventually(func() error {
					// Make an HTTPS request to the server
					resp, err := httpClient.Get("https://127.0.0.1:14534/ping")
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
					}
					return nil
				}, 2*time.Second, 100*time.Millisecond).Should(Succeed())

				// Stop the server
				cancel()

				// Wait for server to stop (with timeout)
				select {
				case <-errChan:
					// Server stopped
				case <-time.After(2 * time.Second):
					Fail("Server did not stop in time")
				}
			})
		})
	})
})
