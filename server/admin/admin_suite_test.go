package admin_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAdminEndpoints(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Admin Endpoints Suite")
}
