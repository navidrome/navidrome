package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Request Helpers", func() {
	var r *http.Request

	Describe("ParamString", func() {
		BeforeEach(func() {
			r = httptest.NewRequest("GET", "/ping?a=123", nil)
		})

		It("returns empty string if param does not exist", func() {
			Expect(ParamString(r, "xx")).To(Equal(""))
		})

		It("returns param as string", func() {
			Expect(ParamString(r, "a")).To(Equal("123"))
		})
	})

	Describe("ParamStringDefault", func() {
		BeforeEach(func() {
			r = httptest.NewRequest("GET", "/ping?a=123", nil)
		})

		It("returns default string if param does not exist", func() {
			Expect(ParamStringDefault(r, "xx", "default_value")).To(Equal("default_value"))
		})

		It("returns param as string", func() {
			Expect(ParamStringDefault(r, "a", "default_value")).To(Equal("123"))
		})
	})

	Describe("ParamStrings", func() {
		BeforeEach(func() {
			r = httptest.NewRequest("GET", "/ping?a=123&a=456", nil)
		})

		It("returns empty array if param does not exist", func() {
			Expect(ParamStrings(r, "xx")).To(BeEmpty())
		})

		It("returns all param occurrences as []string", func() {
			Expect(ParamStrings(r, "a")).To(Equal([]string{"123", "456"}))
		})
	})

	Describe("ParamTime", func() {
		d := time.Date(2002, 8, 9, 12, 11, 13, 1000000, time.Local)
		t := ToMillis(d)
		now := time.Now()
		BeforeEach(func() {
			r = httptest.NewRequest("GET", fmt.Sprintf("/ping?t=%d&inv=abc", t), nil)
		})

		It("returns default time if param does not exist", func() {
			Expect(ParamTime(r, "xx", now)).To(Equal(now))
		})

		It("returns default time if param is an invalid timestamp", func() {
			Expect(ParamTime(r, "inv", now)).To(Equal(now))
		})

		It("returns parsed time", func() {
			Expect(ParamTime(r, "t", now)).To(Equal(d))
		})
	})

	Describe("ParamTimes", func() {
		d1 := time.Date(2002, 8, 9, 12, 11, 13, 1000000, time.Local)
		d2 := time.Date(2002, 8, 9, 12, 13, 56, 0000000, time.Local)
		t1 := ToMillis(d1)
		t2 := ToMillis(d2)
		BeforeEach(func() {
			r = httptest.NewRequest("GET", fmt.Sprintf("/ping?t=%d&t=%d", t1, t2), nil)
		})

		It("returns empty string if param does not exist", func() {
			Expect(ParamTimes(r, "xx")).To(BeEmpty())
		})

		It("returns all param occurrences as []time.Time", func() {
			Expect(ParamTimes(r, "t")).To(Equal([]time.Time{d1, d2}))
		})
		It("returns current time as default if param is invalid", func() {
			now := time.Now()
			r = httptest.NewRequest("GET", "/ping?t=null", nil)
			times := ParamTimes(r, "t")
			Expect(times).To(HaveLen(1))
			Expect(times[0]).To(BeTemporally(">=", now))
		})
	})

	Describe("ParamInt", func() {
		BeforeEach(func() {
			r = httptest.NewRequest("GET", "/ping?i=123&inv=123.45", nil)
		})
		Context("int", func() {
			It("returns default value if param does not exist", func() {
				Expect(ParamInt(r, "xx", 999)).To(Equal(999))
			})

			It("returns default value if param is an invalid int", func() {
				Expect(ParamInt(r, "inv", 999)).To(Equal(999))
			})

			It("returns parsed time", func() {
				Expect(ParamInt(r, "i", 999)).To(Equal(123))
			})
		})
		Context("int64", func() {
			It("returns default value if param does not exist", func() {
				Expect(ParamInt(r, "xx", int64(999))).To(Equal(int64(999)))
			})

			It("returns default value if param is an invalid int", func() {
				Expect(ParamInt(r, "inv", int64(999))).To(Equal(int64(999)))
			})

			It("returns parsed time", func() {
				Expect(ParamInt(r, "i", int64(999))).To(Equal(int64(123)))
			})

		})
	})

	Describe("ParamInts", func() {
		BeforeEach(func() {
			r = httptest.NewRequest("GET", "/ping?i=123&i=456", nil)
		})

		It("returns empty array if param does not exist", func() {
			Expect(ParamInts(r, "xx")).To(BeEmpty())
		})

		It("returns array of occurrences found", func() {
			Expect(ParamInts(r, "i")).To(Equal([]int{123, 456}))
		})
	})

	Describe("ParamBool", func() {
		Context("value is true", func() {
			BeforeEach(func() {
				r = httptest.NewRequest("GET", "/ping?b=true&c=on&d=1&e=True", nil)
			})

			It("parses 'true'", func() {
				Expect(ParamBool(r, "b", false)).To(BeTrue())
			})

			It("parses 'on'", func() {
				Expect(ParamBool(r, "c", false)).To(BeTrue())
			})

			It("parses '1'", func() {
				Expect(ParamBool(r, "d", false)).To(BeTrue())
			})

			It("parses 'True'", func() {
				Expect(ParamBool(r, "e", false)).To(BeTrue())
			})
		})

		Context("value is false", func() {
			BeforeEach(func() {
				r = httptest.NewRequest("GET", "/ping?b=false&c=off&d=0", nil)
			})

			It("returns default value if param does not exist", func() {
				Expect(ParamBool(r, "xx", true)).To(BeTrue())
			})

			It("parses 'false'", func() {
				Expect(ParamBool(r, "b", true)).To(BeFalse())
			})

			It("parses 'off'", func() {
				Expect(ParamBool(r, "c", true)).To(BeFalse())
			})

			It("parses '0'", func() {
				Expect(ParamBool(r, "d", true)).To(BeFalse())
			})
		})
	})
})
