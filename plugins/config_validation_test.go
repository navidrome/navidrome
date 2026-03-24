//go:build !windows

package plugins

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config Validation", func() {
	Describe("ValidateConfig", func() {
		Context("when manifest has no config schema", func() {
			It("returns an error", func() {
				manifest := &Manifest{
					Name:    "test",
					Author:  "test",
					Version: "1.0.0",
				}
				err := ValidateConfig(manifest, `{"key": "value"}`)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no configurable options"))
			})
		})

		Context("when manifest has config schema", func() {
			var manifest *Manifest

			BeforeEach(func() {
				manifest = &Manifest{
					Name:    "test",
					Author:  "test",
					Version: "1.0.0",
					Config: &ConfigDefinition{
						Schema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"apiKey": map[string]any{
									"type":        "string",
									"description": "API key for the service",
									"minLength":   float64(1),
								},
								"timeout": map[string]any{
									"type":    "integer",
									"minimum": float64(1),
									"maximum": float64(300),
								},
								"enabled": map[string]any{
									"type": "boolean",
								},
							},
							"required": []any{"apiKey"},
						},
					},
				}
			})

			It("accepts valid config", func() {
				err := ValidateConfig(manifest, `{"apiKey": "secret123", "timeout": 30}`)
				Expect(err).ToNot(HaveOccurred())
			})

			It("rejects empty config when required fields are missing", func() {
				err := ValidateConfig(manifest, "")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("apiKey"))

				err = ValidateConfig(manifest, "{}")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("apiKey"))
			})

			It("rejects config missing required field", func() {
				err := ValidateConfig(manifest, `{"timeout": 30}`)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("apiKey"))
			})

			It("rejects config with wrong type", func() {
				err := ValidateConfig(manifest, `{"apiKey": "secret", "timeout": "not a number"}`)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("timeout"))
			})

			It("rejects config with value out of range", func() {
				err := ValidateConfig(manifest, `{"apiKey": "secret", "timeout": 500}`)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("timeout"))
			})

			It("rejects config with empty required string", func() {
				err := ValidateConfig(manifest, `{"apiKey": ""}`)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("apiKey"))
			})

			It("rejects invalid JSON", func() {
				err := ValidateConfig(manifest, `{invalid json}`)
				Expect(err).To(HaveOccurred())
				var validationErr *ConfigValidationErrors
				Expect(errors.As(err, &validationErr)).To(BeTrue())
				Expect(validationErr.Errors[0].Message).To(ContainSubstring("invalid JSON"))
			})
		})

		Context("with enum values", func() {
			It("accepts valid enum value", func() {
				manifest := &Manifest{
					Name:    "test",
					Author:  "test",
					Version: "1.0.0",
					Config: &ConfigDefinition{
						Schema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"logLevel": map[string]any{
									"type": "string",
									"enum": []any{"debug", "info", "warn", "error"},
								},
							},
						},
					},
				}
				err := ValidateConfig(manifest, `{"logLevel": "info"}`)
				Expect(err).ToNot(HaveOccurred())
			})

			It("rejects invalid enum value", func() {
				manifest := &Manifest{
					Name:    "test",
					Author:  "test",
					Version: "1.0.0",
					Config: &ConfigDefinition{
						Schema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"logLevel": map[string]any{
									"type": "string",
									"enum": []any{"debug", "info", "warn", "error"},
								},
							},
						},
					},
				}
				err := ValidateConfig(manifest, `{"logLevel": "verbose"}`)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("HasConfigSchema", func() {
		It("returns false when config is nil", func() {
			manifest := &Manifest{
				Name:    "test",
				Author:  "test",
				Version: "1.0.0",
			}
			Expect(manifest.HasConfigSchema()).To(BeFalse())
		})

		It("returns false when schema is nil", func() {
			manifest := &Manifest{
				Name:    "test",
				Author:  "test",
				Version: "1.0.0",
				Config:  &ConfigDefinition{},
			}
			Expect(manifest.HasConfigSchema()).To(BeFalse())
		})

		It("returns true when schema is present", func() {
			manifest := &Manifest{
				Name:    "test",
				Author:  "test",
				Version: "1.0.0",
				Config: &ConfigDefinition{
					Schema: map[string]any{
						"type": "object",
					},
				},
			}
			Expect(manifest.HasConfigSchema()).To(BeTrue())
		})
	})
})
