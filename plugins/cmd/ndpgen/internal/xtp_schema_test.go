package internal

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("XTP Schema Generation", func() {
	parseSchema := func(schema []byte) map[string]any {
		var doc map[string]any
		Expect(yaml.Unmarshal(schema, &doc)).To(Succeed())
		return doc
	}

	Describe("GenerateSchema", func() {
		Context("basic capability with one export", func() {
			var schema []byte

			BeforeEach(func() {
				capability := Capability{
					Name:       "test",
					Doc:        "Test capability",
					SourceFile: "test",
					Methods: []Export{
						{
							ExportName: "test_method",
							Doc:        "Test method does something",
							Input:      NewParam("input", "TestInput"),
							Output:     NewParam("output", "TestOutput"),
						},
					},
					Structs: []StructDef{
						{
							Name: "TestInput",
							Doc:  "Input for test",
							Fields: []FieldDef{
								{Name: "Name", Type: "string", JSONTag: "name", Doc: "The name"},
								{Name: "Count", Type: "int", JSONTag: "count", Doc: "The count"},
							},
						},
						{
							Name: "TestOutput",
							Doc:  "Output for test",
							Fields: []FieldDef{
								{Name: "Result", Type: "string", JSONTag: "result", Doc: "The result"},
							},
						},
					},
				}
				var err error
				schema, err = GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
				Expect(schema).NotTo(BeEmpty())
			})

			It("should validate against XTP JSONSchema", func() {
				Expect(ValidateXTPSchema(schema)).To(Succeed())
			})

			It("should have correct version", func() {
				doc := parseSchema(schema)
				Expect(doc["version"]).To(Equal("v1-draft"))
			})

			It("should include exports with description", func() {
				doc := parseSchema(schema)
				exports := doc["exports"].(map[string]any)
				Expect(exports).To(HaveKey("test_method"))
				method := exports["test_method"].(map[string]any)
				Expect(method["description"]).To(Equal("Test method does something"))
			})

			It("should include schemas for input and output types", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				Expect(schemas).To(HaveKey("TestInput"))
				Expect(schemas).To(HaveKey("TestOutput"))
			})

			It("should define input schema with correct properties", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["TestInput"].(map[string]any)
				// Per XTP spec, ObjectSchema does NOT have a type field - only properties, required, description
				Expect(input).NotTo(HaveKey("type"))
				props := input["properties"].(map[string]any)
				Expect(props).To(HaveKey("name"))
				Expect(props).To(HaveKey("count"))
			})

			It("should mark non-pointer, non-omitempty fields as required", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["TestInput"].(map[string]any)
				required := input["required"].([]any)
				Expect(required).To(ContainElement("name"))
				Expect(required).To(ContainElement("count"))
			})
		})

		Context("capability with pointer fields (nullable)", func() {
			var schema []byte

			BeforeEach(func() {
				capability := Capability{
					Name:       "nullable_test",
					SourceFile: "nullable_test",
					Methods: []Export{
						{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
					},
					Structs: []StructDef{
						{
							Name: "Input",
							Fields: []FieldDef{
								{Name: "Required", Type: "string", JSONTag: "required"},
								{Name: "Optional", Type: "*string", JSONTag: "optional,omitempty", OmitEmpty: true},
							},
						},
						{
							Name: "Output",
							Fields: []FieldDef{
								{Name: "Value", Type: "string", JSONTag: "value"},
							},
						},
					},
				}
				var err error
				schema, err = GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate against XTP JSONSchema", func() {
				Expect(ValidateXTPSchema(schema)).To(Succeed())
			})

			It("should not mark required field as nullable", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["Input"].(map[string]any)
				props := input["properties"].(map[string]any)
				requiredField := props["required"].(map[string]any)
				Expect(requiredField).NotTo(HaveKey("nullable"))
			})

			It("should mark optional pointer field as nullable", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["Input"].(map[string]any)
				props := input["properties"].(map[string]any)
				optionalField := props["optional"].(map[string]any)
				Expect(optionalField["nullable"]).To(BeTrue())
			})

			It("should only include non-pointer fields in required array", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["Input"].(map[string]any)
				required := input["required"].([]any)
				Expect(required).To(ContainElement("required"))
				Expect(required).NotTo(ContainElement("optional"))
			})
		})

		Context("capability with enum", func() {
			var schema []byte

			BeforeEach(func() {
				capability := Capability{
					Name:       "enum_test",
					SourceFile: "enum_test",
					Methods: []Export{
						{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
					},
					Structs: []StructDef{
						{
							Name: "Input",
							Fields: []FieldDef{
								{Name: "Status", Type: "Status", JSONTag: "status"},
							},
						},
						{
							Name: "Output",
							Fields: []FieldDef{
								{Name: "Value", Type: "string", JSONTag: "value"},
							},
						},
					},
					TypeAliases: []TypeAlias{
						{Name: "Status", Type: "string", Doc: "Status type"},
					},
					Consts: []ConstGroup{
						{
							Type: "Status",
							Values: []ConstDef{
								{Name: "StatusPending", Value: `"pending"`},
								{Name: "StatusActive", Value: `"active"`},
								{Name: "StatusDone", Value: `"done"`},
							},
						},
					},
				}
				var err error
				schema, err = GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate against XTP JSONSchema", func() {
				Expect(ValidateXTPSchema(schema)).To(Succeed())
			})

			It("should define enum type with correct values", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				Expect(schemas).To(HaveKey("Status"))
				status := schemas["Status"].(map[string]any)
				Expect(status["type"]).To(Equal("string"))
				enum := status["enum"].([]any)
				Expect(enum).To(ConsistOf("pending", "active", "done"))
			})

			It("should use $ref for enum field in struct", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["Input"].(map[string]any)
				props := input["properties"].(map[string]any)
				statusRef := props["status"].(map[string]any)
				Expect(statusRef["$ref"]).To(Equal("#/components/schemas/Status"))
			})
		})

		Context("capability with array types", func() {
			var schema []byte

			BeforeEach(func() {
				capability := Capability{
					Name:       "array_test",
					SourceFile: "array_test",
					Methods: []Export{
						{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
					},
					Structs: []StructDef{
						{
							Name: "Input",
							Fields: []FieldDef{
								{Name: "Tags", Type: "[]string", JSONTag: "tags"},
								{Name: "Items", Type: "[]Item", JSONTag: "items"},
							},
						},
						{
							Name: "Output",
							Fields: []FieldDef{
								{Name: "Value", Type: "string", JSONTag: "value"},
							},
						},
						{
							Name: "Item",
							Fields: []FieldDef{
								{Name: "ID", Type: "string", JSONTag: "id"},
							},
						},
					},
				}
				var err error
				schema, err = GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate against XTP JSONSchema", func() {
				Expect(ValidateXTPSchema(schema)).To(Succeed())
			})

			It("should define string array with primitive type", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["Input"].(map[string]any)
				props := input["properties"].(map[string]any)
				tags := props["tags"].(map[string]any)
				Expect(tags["type"]).To(Equal("array"))
				tagItems := tags["items"].(map[string]any)
				Expect(tagItems["type"]).To(Equal("string"))
			})

			It("should define struct array with $ref", func() {
				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["Input"].(map[string]any)
				props := input["properties"].(map[string]any)
				items := props["items"].(map[string]any)
				Expect(items["type"]).To(Equal("array"))
				itemItems := items["items"].(map[string]any)
				Expect(itemItems["$ref"]).To(Equal("#/components/schemas/Item"))
			})
		})

		Context("capability with []byte field", func() {
			It("should map []byte to string with byte format, not array", func() {
				capability := Capability{
					Name:       "byte_test",
					SourceFile: "byte_test",
					Methods: []Export{
						{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
					},
					Structs: []StructDef{
						{
							Name: "Input",
							Fields: []FieldDef{
								{Name: "Data", Type: "[]byte", JSONTag: "data"},
							},
						},
						{
							Name: "Output",
							Fields: []FieldDef{
								{Name: "Value", Type: "string", JSONTag: "value"},
							},
						},
					},
				}
				schema, err := GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
				Expect(ValidateXTPSchema(schema)).To(Succeed())

				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["Input"].(map[string]any)
				props := input["properties"].(map[string]any)
				data := props["data"].(map[string]any)
				Expect(data["type"]).To(Equal("string"))
				Expect(data["format"]).To(Equal("byte"))
				Expect(data).NotTo(HaveKey("items"))
			})
		})

		Context("capability with nullable ref", func() {
			It("should mark pointer to enum as nullable with $ref", func() {
				capability := Capability{
					Name:       "nullable_ref_test",
					SourceFile: "nullable_ref_test",
					Methods: []Export{
						{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
					},
					Structs: []StructDef{
						{
							Name: "Input",
							Fields: []FieldDef{
								{Name: "Value", Type: "string", JSONTag: "value"},
							},
						},
						{
							Name: "Output",
							Fields: []FieldDef{
								{Name: "Status", Type: "*ErrorType", JSONTag: "status,omitempty", OmitEmpty: true},
							},
						},
					},
					TypeAliases: []TypeAlias{
						{Name: "ErrorType", Type: "string"},
					},
					Consts: []ConstGroup{
						{
							Type: "ErrorType",
							Values: []ConstDef{
								{Name: "ErrorNone", Value: `"none"`},
								{Name: "ErrorFatal", Value: `"fatal"`},
							},
						},
					},
				}
				schema, err := GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())

				// Validate against XTP JSONSchema
				Expect(ValidateXTPSchema(schema)).To(Succeed())

				doc := parseSchema(schema)
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				output := schemas["Output"].(map[string]any)
				props := output["properties"].(map[string]any)
				status := props["status"].(map[string]any)
				Expect(status["$ref"]).To(Equal("#/components/schemas/ErrorType"))
				Expect(status["nullable"]).To(BeTrue())
			})
		})
	})

	Describe("goTypeToXTPTypeAndFormat", func() {
		DescribeTable("should convert Go types to XTP types",
			func(goType, wantType, wantFormat string) {
				gotType, gotFormat := goTypeToXTPTypeAndFormat(goType)
				Expect(gotType).To(Equal(wantType))
				Expect(gotFormat).To(Equal(wantFormat))
			},
			Entry("string", "string", "string", ""),
			Entry("int", "int", "integer", "int32"),
			Entry("int32", "int32", "integer", "int32"),
			Entry("int64", "int64", "integer", "int64"),
			Entry("float32", "float32", "number", "float"),
			Entry("float64", "float64", "number", "float"),
			Entry("bool", "bool", "boolean", ""),
			Entry("[]byte", "[]byte", "string", "byte"),
			Entry("unknown types default to object", "CustomType", "object", ""),
		)
	})

	Describe("cleanDocForYAML", func() {
		DescribeTable("should clean documentation strings",
			func(doc, want string) {
				Expect(cleanDocForYAML(doc)).To(Equal(want))
			},
			Entry("empty", "", ""),
			Entry("single line", "Simple description", "Simple description"),
			Entry("multiline", "First line\nSecond line", "First line\nSecond line"),
			Entry("trailing newline", "Description\n", "Description"),
			Entry("whitespace", "  Description  ", "Description"),
		)
	})

	Describe("isPrimitiveGoType", func() {
		DescribeTable("should identify primitive Go types",
			func(goType string, want bool) {
				Expect(isPrimitiveGoType(goType)).To(Equal(want))
			},
			Entry("bool", "bool", true),
			Entry("string", "string", true),
			Entry("int", "int", true),
			Entry("int32", "int32", true),
			Entry("int64", "int64", true),
			Entry("float32", "float32", true),
			Entry("float64", "float64", true),
			Entry("[]byte", "[]byte", true),
			Entry("custom type", "CustomType", false),
			Entry("struct type", "MyStruct", false),
			Entry("slice of string", "[]string", false),
			Entry("map type", "map[string]int", false),
		)
	})

	Describe("GenerateSchema with primitive output types", func() {
		inputStruct := StructDef{
			Name:   "Input",
			Fields: []FieldDef{{Name: "ID", Type: "string", JSONTag: "id"}},
		}

		Context("export with primitive string output", func() {
			It("should use type instead of $ref and validate against XTP JSONSchema", func() {
				capability := Capability{
					Name:       "test",
					SourceFile: "test",
					Methods: []Export{
						{ExportName: "get_name", Input: NewParam("input", "Input"), Output: NewParam("output", "string")},
					},
					Structs: []StructDef{inputStruct},
				}
				schema, err := GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
				Expect(schema).NotTo(BeEmpty())
				Expect(ValidateXTPSchema(schema)).To(Succeed())

				doc := parseSchema(schema)
				exports := doc["exports"].(map[string]any)
				method := exports["get_name"].(map[string]any)
				output := method["output"].(map[string]any)
				Expect(output["type"]).To(Equal("string"))
				Expect(output).NotTo(HaveKey("$ref"))
				Expect(output["contentType"]).To(Equal("application/json"))
			})
		})

		Context("export with primitive bool output", func() {
			It("should use boolean type and validate against XTP JSONSchema", func() {
				capability := Capability{
					Name:       "test",
					SourceFile: "test",
					Methods: []Export{
						{ExportName: "is_valid", Input: NewParam("input", "Input"), Output: NewParam("output", "bool")},
					},
					Structs: []StructDef{inputStruct},
				}
				schema, err := GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
				Expect(ValidateXTPSchema(schema)).To(Succeed())

				doc := parseSchema(schema)
				exports := doc["exports"].(map[string]any)
				method := exports["is_valid"].(map[string]any)
				output := method["output"].(map[string]any)
				Expect(output["type"]).To(Equal("boolean"))
				Expect(output).NotTo(HaveKey("$ref"))
			})
		})

		Context("export with primitive int output", func() {
			It("should use integer type and validate against XTP JSONSchema", func() {
				capability := Capability{
					Name:       "test",
					SourceFile: "test",
					Methods: []Export{
						{ExportName: "get_count", Input: NewParam("input", "Input"), Output: NewParam("output", "int32")},
					},
					Structs: []StructDef{inputStruct},
				}
				schema, err := GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
				Expect(ValidateXTPSchema(schema)).To(Succeed())

				doc := parseSchema(schema)
				exports := doc["exports"].(map[string]any)
				method := exports["get_count"].(map[string]any)
				output := method["output"].(map[string]any)
				Expect(output["type"]).To(Equal("integer"))
				Expect(output).NotTo(HaveKey("$ref"))
			})
		})

		Context("export with pointer to primitive output", func() {
			It("should strip pointer and use primitive type and validate against XTP JSONSchema", func() {
				capability := Capability{
					Name:       "test",
					SourceFile: "test",
					Methods: []Export{
						{ExportName: "get_optional_string", Input: NewParam("input", "Input"), Output: NewParam("output", "*string")},
					},
					Structs: []StructDef{inputStruct},
				}
				schema, err := GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
				Expect(ValidateXTPSchema(schema)).To(Succeed())

				doc := parseSchema(schema)
				exports := doc["exports"].(map[string]any)
				method := exports["get_optional_string"].(map[string]any)
				output := method["output"].(map[string]any)
				Expect(output["type"]).To(Equal("string"))
				Expect(output).NotTo(HaveKey("$ref"))
			})
		})

		Context("export with struct output", func() {
			It("should still use $ref and validate against XTP JSONSchema", func() {
				capability := Capability{
					Name:       "test",
					SourceFile: "test",
					Methods: []Export{
						{ExportName: "get_result", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
					},
					Structs: []StructDef{
						inputStruct,
						{Name: "Output", Fields: []FieldDef{{Name: "Value", Type: "string", JSONTag: "value"}}},
					},
				}
				schema, err := GenerateSchema(capability)
				Expect(err).NotTo(HaveOccurred())
				Expect(ValidateXTPSchema(schema)).To(Succeed())

				doc := parseSchema(schema)
				exports := doc["exports"].(map[string]any)
				method := exports["get_result"].(map[string]any)
				output := method["output"].(map[string]any)
				Expect(output["$ref"]).To(Equal("#/components/schemas/Output"))
				Expect(output).NotTo(HaveKey("type"))
			})
		})
	})

	Describe("collectUsedTypes", func() {
		getSchemas := func(schema []byte) map[string]any {
			doc := parseSchema(schema)
			components, hasComponents := doc["components"].(map[string]any)
			if !hasComponents {
				return make(map[string]any)
			}
			schemas, ok := components["schemas"].(map[string]any)
			if !ok {
				return make(map[string]any)
			}
			return schemas
		}

		It("should only include types referenced by exports", func() {
			capability := Capability{
				Name:       "test",
				SourceFile: "test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "UsedInput"), Output: NewParam("output", "UsedOutput")},
				},
				Structs: []StructDef{
					{Name: "UsedInput", Fields: []FieldDef{{Name: "ID", Type: "string", JSONTag: "id"}}},
					{Name: "UsedOutput", Fields: []FieldDef{{Name: "Value", Type: "string", JSONTag: "value"}}},
					{Name: "UnusedStruct", Fields: []FieldDef{{Name: "Foo", Type: "string", JSONTag: "foo"}}},
				},
			}
			schema, err := GenerateSchema(capability)
			Expect(err).NotTo(HaveOccurred())
			Expect(ValidateXTPSchema(schema)).To(Succeed())

			schemas := getSchemas(schema)
			Expect(schemas).To(HaveKey("UsedInput"))
			Expect(schemas).To(HaveKey("UsedOutput"))
			Expect(schemas).NotTo(HaveKey("UnusedStruct"))
		})

		It("should include transitively referenced types", func() {
			capability := Capability{
				Name:       "test",
				SourceFile: "test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
				},
				Structs: []StructDef{
					{Name: "Input", Fields: []FieldDef{{Name: "ID", Type: "string", JSONTag: "id"}}},
					{Name: "Output", Fields: []FieldDef{{Name: "Nested", Type: "NestedType", JSONTag: "nested"}}},
					{Name: "NestedType", Fields: []FieldDef{{Name: "Value", Type: "string", JSONTag: "value"}}},
				},
			}
			schema, err := GenerateSchema(capability)
			Expect(err).NotTo(HaveOccurred())
			Expect(ValidateXTPSchema(schema)).To(Succeed())

			schemas := getSchemas(schema)
			Expect(schemas).To(HaveKey("Input"))
			Expect(schemas).To(HaveKey("Output"))
			Expect(schemas).To(HaveKey("NestedType"))
		})

		It("should include array element types", func() {
			capability := Capability{
				Name:       "test",
				SourceFile: "test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
				},
				Structs: []StructDef{
					{Name: "Input", Fields: []FieldDef{{Name: "ID", Type: "string", JSONTag: "id"}}},
					{Name: "Output", Fields: []FieldDef{{Name: "Items", Type: "[]Item", JSONTag: "items"}}},
					{Name: "Item", Fields: []FieldDef{{Name: "Name", Type: "string", JSONTag: "name"}}},
				},
			}
			schema, err := GenerateSchema(capability)
			Expect(err).NotTo(HaveOccurred())
			Expect(ValidateXTPSchema(schema)).To(Succeed())

			schemas := getSchemas(schema)
			Expect(schemas).To(HaveKey("Input"))
			Expect(schemas).To(HaveKey("Output"))
			Expect(schemas).To(HaveKey("Item"))
		})

		It("should include pointer types", func() {
			capability := Capability{
				Name:       "test",
				SourceFile: "test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
				},
				Structs: []StructDef{
					{Name: "Input", Fields: []FieldDef{{Name: "ID", Type: "string", JSONTag: "id"}}},
					{Name: "Output", Fields: []FieldDef{{Name: "Optional", Type: "*OptionalType", JSONTag: "optional"}}},
					{Name: "OptionalType", Fields: []FieldDef{{Name: "Value", Type: "string", JSONTag: "value"}}},
				},
			}
			schema, err := GenerateSchema(capability)
			Expect(err).NotTo(HaveOccurred())
			Expect(ValidateXTPSchema(schema)).To(Succeed())

			schemas := getSchemas(schema)
			Expect(schemas).To(HaveKey("Input"))
			Expect(schemas).To(HaveKey("Output"))
			Expect(schemas).To(HaveKey("OptionalType"))
		})

		It("should exclude primitive output types from schema", func() {
			capability := Capability{
				Name:       "test",
				SourceFile: "test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "string")},
				},
				Structs: []StructDef{
					{Name: "Input", Fields: []FieldDef{{Name: "ID", Type: "string", JSONTag: "id"}}},
				},
			}
			schema, err := GenerateSchema(capability)
			Expect(err).NotTo(HaveOccurred())
			Expect(ValidateXTPSchema(schema)).To(Succeed())

			schemas := getSchemas(schema)
			Expect(schemas).To(HaveKey("Input"))
		})
	})

	Describe("GenerateSchema with shared aliases", func() {
		It("inlines shared-alias shapes as schema components", func() {
			cap := Capability{
				Name: "scrobbler", Interface: "Scrobbler", Required: true,
				Methods: []Export{{Name: "NowPlaying", ExportName: "nd_scrobbler_now_playing",
					Input: Param{Name: "input", Type: "NowPlayingRequest"}}},
				Structs: []StructDef{{Name: "NowPlayingRequest", Fields: []FieldDef{
					{Name: "Track", Type: "TrackInfo", JSONTag: "track"}}}},
				SharedAliases: []SharedAlias{{
					Name: "TrackInfo", Target: "types.TrackInfo",
					Def: StructDef{Name: "TrackInfo", Fields: []FieldDef{
						{Name: "Title", Type: "string", JSONTag: "title"}}},
				}},
			}
			out, err := GenerateSchema(cap)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(ContainSubstring("TrackInfo:"))
			Expect(string(out)).To(ContainSubstring("title:"))
		})

		It("names the shared component by its canonical type for qualified types.X fields", func() {
			cap := Capability{
				Name: "scrobbler", Interface: "Scrobbler", Required: true,
				Methods: []Export{{Name: "NowPlaying", ExportName: "nd_scrobbler_now_playing",
					Input: Param{Name: "input", Type: "NowPlayingRequest"}}},
				Structs: []StructDef{{Name: "NowPlayingRequest", Fields: []FieldDef{
					{Name: "Track", Type: "types.Track", JSONTag: "track"}}}},
				SharedAliases: []SharedAlias{{
					Name: "TrackInfo", Target: "types.Track",
					Def: StructDef{Name: "Track", Fields: []FieldDef{
						{Name: "Title", Type: "string", JSONTag: "title"}}},
				}},
			}
			out, err := GenerateSchema(cap)
			Expect(err).NotTo(HaveOccurred())
			s := string(out)
			// Component is named by the canonical type (Track), not the deprecated alias.
			Expect(s).To(ContainSubstring("Track:"))
			Expect(s).NotTo(ContainSubstring("TrackInfo:"))
			// The field $ref points at the canonical component.
			Expect(s).To(ContainSubstring("$ref: '#/components/schemas/Track'"))
			Expect(s).To(ContainSubstring("title:"))
		})

		It("points an alias-named field at the canonical component for a renamed alias", func() {
			cap := Capability{
				Name: "demo", Interface: "Demo", Required: true,
				Methods: []Export{{Name: "Play", ExportName: "nd_demo_play",
					Input: Param{Name: "input", Type: "PlayRequest"}}},
				Structs: []StructDef{{Name: "PlayRequest", Fields: []FieldDef{
					// Field is typed with the deprecated alias name, not the canonical types.Track.
					{Name: "Track", Type: "TrackInfo", JSONTag: "track"}}}},
				SharedAliases: []SharedAlias{{
					Name: "TrackInfo", Target: "types.Track",
					Def: StructDef{Name: "Track", Fields: []FieldDef{
						{Name: "Title", Type: "string", JSONTag: "title"}}},
				}},
			}
			out, err := GenerateSchema(cap)
			Expect(err).NotTo(HaveOccurred())
			s := string(out)
			// The component is emitted under the canonical name, and the field $ref must
			// point at it — not at a non-existent TrackInfo component (dangling reference).
			Expect(s).To(ContainSubstring("Track:"))
			Expect(s).To(ContainSubstring("$ref: '#/components/schemas/Track'"))
			Expect(s).NotTo(ContainSubstring("$ref: '#/components/schemas/TrackInfo'"))
		})

		It("points an alias-named export input/output at the canonical component", func() {
			cap := Capability{
				Name: "demo", Interface: "Demo", Required: true,
				// The method takes/returns the deprecated alias name directly.
				Methods: []Export{{Name: "Play", ExportName: "nd_demo_play",
					Input:  Param{Name: "input", Type: "TrackInfo"},
					Output: Param{Name: "output", Type: "TrackInfo"}}},
				SharedAliases: []SharedAlias{{
					Name: "TrackInfo", Target: "types.Track",
					Def: StructDef{Name: "Track", Fields: []FieldDef{
						{Name: "Title", Type: "string", JSONTag: "title"}}},
				}},
			}
			out, err := GenerateSchema(cap)
			Expect(err).NotTo(HaveOccurred())
			s := string(out)
			// Export $ref must resolve to the canonical component, not a missing TrackInfo.
			Expect(s).To(ContainSubstring("$ref: '#/components/schemas/Track'"))
			Expect(s).NotTo(ContainSubstring("$ref: '#/components/schemas/TrackInfo'"))
		})

		It("inlines a directly-referenced shared type that has no deprecated alias", func() {
			cap := Capability{
				Name: "scrobbler", Interface: "Scrobbler", Required: true,
				Methods: []Export{{Name: "NowPlaying", ExportName: "nd_scrobbler_now_playing",
					Input: Param{Name: "input", Type: "NowPlayingRequest"}}},
				Structs: []StructDef{{Name: "NowPlayingRequest", Fields: []FieldDef{
					{Name: "Song", Type: "types.SongRef", JSONTag: "song"}}}},
				// No SharedAliases: the field references the canonical type directly.
				SharedTypes: []StructDef{{Name: "SongRef", Fields: []FieldDef{
					{Name: "Name", Type: "string", JSONTag: "name"}}}},
			}
			out, err := GenerateSchema(cap)
			Expect(err).NotTo(HaveOccurred())
			s := string(out)
			Expect(s).To(ContainSubstring("SongRef:"))
			Expect(s).To(ContainSubstring("$ref: '#/components/schemas/SongRef'"))
			Expect(s).To(ContainSubstring("name:"))
		})
	})

	Describe("GenerateSchema enum filtering", func() {
		It("should only include enums that are actually used by exports", func() {
			capability := Capability{
				Name:       "test",
				SourceFile: "test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
				},
				Structs: []StructDef{
					{
						Name:   "Input",
						Fields: []FieldDef{{Name: "Status", Type: "UsedStatus", JSONTag: "status"}},
					},
					{
						Name:   "Output",
						Fields: []FieldDef{{Name: "Value", Type: "string", JSONTag: "value"}},
					},
				},
				TypeAliases: []TypeAlias{
					{Name: "UsedStatus", Type: "string"},
					{Name: "UnusedStatus", Type: "string"},
				},
				Consts: []ConstGroup{
					{
						Type: "UsedStatus",
						Values: []ConstDef{
							{Name: "StatusActive", Value: `"active"`},
							{Name: "StatusInactive", Value: `"inactive"`},
						},
					},
					{
						Type: "UnusedStatus",
						Values: []ConstDef{
							{Name: "UnusedPending", Value: `"pending"`},
						},
					},
				},
			}

			schema, err := GenerateSchema(capability)
			Expect(err).NotTo(HaveOccurred())
			Expect(ValidateXTPSchema(schema)).To(Succeed())

			doc := parseSchema(schema)
			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)

			// UsedStatus should be included because it's referenced by Input
			Expect(schemas).To(HaveKey("UsedStatus"))
			usedStatus := schemas["UsedStatus"].(map[string]any)
			Expect(usedStatus["type"]).To(Equal("string"))
			enum := usedStatus["enum"].([]any)
			Expect(enum).To(ConsistOf("active", "inactive"))

			// UnusedStatus should NOT be included
			Expect(schemas).NotTo(HaveKey("UnusedStatus"))
		})
	})
})
