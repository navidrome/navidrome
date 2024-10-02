package req_test

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/navidrome/navidrome/utils/req"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Request Helpers Suite")
}

var _ = Describe("Request Helpers", func() {
	var r *req.Values

	Describe("ParamString", func() {
		BeforeEach(func() {
			r = req.Params(httptest.NewRequest("GET", "/ping?a=123", nil))
		})

		It("returns param as string", func() {
			Expect(r.String("a")).To(Equal("123"))
		})

		It("returns empty string if param does not exist", func() {
			v, err := r.String("NON_EXISTENT_PARAM")
			Expect(err).To(MatchError(req.ErrMissingParam))
			Expect(err.Error()).To(ContainSubstring("NON_EXISTENT_PARAM"))
			Expect(v).To(BeEmpty())
		})
	})

	Describe("ParamStringDefault", func() {
		BeforeEach(func() {
			r = req.Params(httptest.NewRequest("GET", "/ping?a=123", nil))
		})

		It("returns param as string", func() {
			Expect(r.StringOr("a", "default_value")).To(Equal("123"))
		})

		It("returns default string if param does not exist", func() {
			Expect(r.StringOr("xx", "default_value")).To(Equal("default_value"))
		})
	})

	Describe("ParamStrings", func() {
		BeforeEach(func() {
			r = req.Params(httptest.NewRequest("GET", "/ping?a=123&a=456", nil))
		})

		It("returns all param occurrences as []string", func() {
			Expect(r.Strings("a")).To(Equal([]string{"123", "456"}))
		})

		It("returns empty array if param does not exist", func() {
			v, err := r.Strings("xx")
			Expect(err).To(MatchError(req.ErrMissingParam))
			Expect(v).To(BeEmpty())
		})
	})

	Describe("ParamTime", func() {
		d := time.Date(2002, 8, 9, 12, 11, 13, 1000000, time.Local)
		t := d.UnixMilli()
		now := time.Now()
		BeforeEach(func() {
			r = req.Params(httptest.NewRequest("GET", fmt.Sprintf("/ping?t=%d&inv=abc", t), nil))
		})

		It("returns parsed time", func() {
			Expect(r.TimeOr("t", now)).To(Equal(d))
		})

		It("returns default time if param does not exist", func() {
			Expect(r.TimeOr("xx", now)).To(Equal(now))
		})

		It("returns default time if param is an invalid timestamp", func() {
			Expect(r.TimeOr("inv", now)).To(Equal(now))
		})
	})

	Describe("ParamTimes", func() {
		d1 := time.Date(2002, 8, 9, 12, 11, 13, 1000000, time.Local)
		d2 := time.Date(2002, 8, 9, 12, 13, 56, 0000000, time.Local)
		t1 := d1.UnixMilli()
		t2 := d2.UnixMilli()
		BeforeEach(func() {
			r = req.Params(httptest.NewRequest("GET", fmt.Sprintf("/ping?t=%d&t=%d", t1, t2), nil))
		})

		It("returns all param occurrences as []time.Time", func() {
			Expect(r.Times("t")).To(Equal([]time.Time{d1, d2}))
		})

		It("returns empty string if param does not exist", func() {
			v, err := r.Times("xx")
			Expect(err).To(MatchError(req.ErrMissingParam))
			Expect(v).To(BeEmpty())
		})

		It("returns current time as default if param is invalid", func() {
			now := time.Now()
			r = req.Params(httptest.NewRequest("GET", "/ping?t=null", nil))
			times, err := r.Times("t")
			Expect(err).ToNot(HaveOccurred())
			Expect(times).To(HaveLen(1))
			Expect(times[0]).To(BeTemporally(">=", now))
		})
	})

	Describe("ParamInt", func() {
		BeforeEach(func() {
			r = req.Params(httptest.NewRequest("GET", "/ping?i=123&inv=123.45", nil))
		})
		Context("int", func() {
			It("returns parsed int", func() {
				Expect(r.IntOr("i", 999)).To(Equal(123))
			})

			It("returns default value if param does not exist", func() {
				Expect(r.IntOr("xx", 999)).To(Equal(999))
			})

			It("returns default value if param is an invalid int", func() {
				Expect(r.IntOr("inv", 999)).To(Equal(999))
			})

			It("returns error if param is an invalid int", func() {
				_, err := r.Int("inv")
				Expect(err).To(MatchError(req.ErrInvalidParam))
			})
		})
		Context("int64", func() {
			It("returns parsed int64", func() {
				Expect(r.Int64Or("i", 999)).To(Equal(int64(123)))
			})

			It("returns default value if param does not exist", func() {
				Expect(r.Int64Or("xx", 999)).To(Equal(int64(999)))
			})

			It("returns default value if param is an invalid int", func() {
				Expect(r.Int64Or("inv", 999)).To(Equal(int64(999)))
			})

			It("returns error if param is an invalid int", func() {
				_, err := r.Int64("inv")
				Expect(err).To(MatchError(req.ErrInvalidParam))
			})
		})
	})

	Describe("ParamInts", func() {
		BeforeEach(func() {
			r = req.Params(httptest.NewRequest("GET", "/ping?i=123&i=456", nil))
		})

		It("returns array of occurrences found", func() {
			Expect(r.Ints("i")).To(Equal([]int{123, 456}))
		})

		It("returns empty array if param does not exist", func() {
			v, err := r.Ints("xx")
			Expect(err).To(MatchError(req.ErrMissingParam))
			Expect(v).To(BeEmpty())
		})
	})

	Describe("ParamBool", func() {
		Context("value is true", func() {
			BeforeEach(func() {
				r = req.Params(httptest.NewRequest("GET", "/ping?b=true&c=on&d=1&e=True", nil))
			})

			It("parses 'true'", func() {
				Expect(r.BoolOr("b", false)).To(BeTrue())
			})

			It("parses 'on'", func() {
				Expect(r.BoolOr("c", false)).To(BeTrue())
			})

			It("parses '1'", func() {
				Expect(r.BoolOr("d", false)).To(BeTrue())
			})

			It("parses 'True'", func() {
				Expect(r.BoolOr("e", false)).To(BeTrue())
			})
		})

		Context("value is false", func() {
			BeforeEach(func() {
				r = req.Params(httptest.NewRequest("GET", "/ping?b=false&c=off&d=0", nil))
			})

			It("parses 'false'", func() {
				Expect(r.BoolOr("b", true)).To(BeFalse())
			})

			It("parses 'off'", func() {
				Expect(r.BoolOr("c", true)).To(BeFalse())
			})

			It("parses '0'", func() {
				Expect(r.BoolOr("d", true)).To(BeFalse())
			})

			It("returns default value if param does not exist", func() {
				Expect(r.BoolOr("xx", true)).To(BeTrue())
			})
		})
	})
})
