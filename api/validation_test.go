package api_test
//
//import (
//	"encoding/xml"
//	"fmt"
//	"testing"
//
//	"context"
//
//	"github.com/astaxie/beego"
//	"github.com/cloudsonic/sonic-server/api"
//	"github.com/cloudsonic/sonic-server/api/responses"
//	"github.com/cloudsonic/sonic-server/tests"
//	. "github.com/smartystreets/goconvey/convey"
//)
//
//func TestCheckParams(t *testing.T) {
//	tests.Init(t, false)
//
//	_, w := Get("/rest/ping.view", "TestCheckParams")
//
//	Convey("Subject: CheckParams\n", t, func() {
//		Convey("Status code should be 200", func() {
//			So(w.Code, ShouldEqual, 200)
//		})
//		Convey("The errorCode should be 10", func() {
//			So(w.Body.String(), ShouldContainSubstring, `error code="10" message=`)
//		})
//		Convey("The status should be 'fail'", func() {
//			v := responses.Subsonic{}
//			xml.Unmarshal(w.Body.Bytes(), &v)
//			So(v.Status, ShouldEqual, "fail")
//		})
//	})
//}
//
//func TestAuthentication(t *testing.T) {
//	tests.Init(t, false)
//
//	Convey("Subject: Authentication", t, func() {
//		_, w := Get("/rest/ping.view?u=INVALID&p=INVALID&c=test&v=1.0.0", "TestAuthentication")
//		Convey("Status code should be 200", func() {
//			So(w.Code, ShouldEqual, 200)
//		})
//		Convey("The errorCode should be 10", func() {
//			So(w.Body.String(), ShouldContainSubstring, `error code="40" message=`)
//		})
//		Convey("The status should be 'fail'", func() {
//			v := responses.Subsonic{}
//			xml.Unmarshal(w.Body.Bytes(), &v)
//			So(v.Status, ShouldEqual, "fail")
//		})
//	})
//	Convey("Subject: Authentication Valid", t, func() {
//		_, w := Get("/rest/ping.view?u=deluan&p=wordpass&c=test&v=1.0.0", "TestAuthentication")
//		Convey("The status should be 'ok'", func() {
//			v := responses.Subsonic{}
//			xml.Unmarshal(w.Body.Bytes(), &v)
//			So(v.Status, ShouldEqual, "ok")
//		})
//	})
//	Convey("Subject: Password encoded", t, func() {
//		_, w := Get("/rest/ping.view?u=deluan&p=enc:776f726470617373&c=test&v=1.0.0", "TestAuthentication")
//		Convey("The status should be 'ok'", func() {
//			v := responses.Subsonic{}
//			xml.Unmarshal(w.Body.Bytes(), &v)
//			So(v.Status, ShouldEqual, "ok")
//		})
//	})
//	Convey("Subject: Token-based authentication", t, func() {
//		salt := "retnlmjetrymazgkt"
//		token := "23b342970e25c7928831c3317edd0b67"
//		_, w := Get(fmt.Sprintf("/rest/ping.view?u=deluan&s=%s&t=%s&c=test&v=1.0.0", salt, token), "TestAuthentication")
//		Convey("The status should be 'ok'", func() {
//			v := responses.Subsonic{}
//			xml.Unmarshal(w.Body.Bytes(), &v)
//			So(v.Status, ShouldEqual, "ok")
//		})
//	})
//}
//
//type mockController struct {
//	api.BaseAPIController
//}
//
//func (c *mockController) Get() {
//	actualContext = c.Ctx.Input.GetData("context").(context.Context)
//	c.Ctx.WriteString("OK")
//}
//
//var actualContext context.Context
//
//func TestContext(t *testing.T) {
//	tests.Init(t, false)
//	beego.Router("/rest/mocktest", &mockController{})
//
//	Convey("Subject: Context", t, func() {
//		_, w := GetWithHeader("/rest/mocktest?u=deluan&p=wordpass&c=testClient&v=1.0.0", "X-Request-Id", "123123", "TestContext")
//		Convey("The status should be 'OK'", func() {
//			resp := string(w.Body.Bytes())
//			So(resp, ShouldEqual, "OK")
//		})
//		Convey("user should be set", func() {
//			So(actualContext.Value("user"), ShouldEqual, "deluan")
//		})
//		Convey("client should be set", func() {
//			So(actualContext.Value("client"), ShouldEqual, "testClient")
//		})
//		Convey("version should be set", func() {
//			So(actualContext.Value("version"), ShouldEqual, "1.0.0")
//		})
//		Convey("context should be set", func() {
//			So(actualContext.Value("requestId"), ShouldEqual, "123123")
//		})
//	})
//}
