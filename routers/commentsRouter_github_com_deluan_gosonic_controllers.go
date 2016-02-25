package routers

import (
	"github.com/astaxie/beego"
)

func init() {

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:GetLicenseController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:GetLicenseController"],
		beego.ControllerComments{
			"Get",
			`/rest/getLicense.view`,
			[]string{"get"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:GetMusicFoldersController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:GetMusicFoldersController"],
		beego.ControllerComments{
			"Get",
			`/rest/getMusicFolders.view`,
			[]string{"get"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:MainController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:MainController"],
		beego.ControllerComments{
			"Get",
			`/`,
			[]string{"get"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"],
		beego.ControllerComments{
			"Post",
			`/`,
			[]string{"post"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"],
		beego.ControllerComments{
			"Get",
			`/:objectId`,
			[]string{"get"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"],
		beego.ControllerComments{
			"GetAll",
			`/`,
			[]string{"get"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"],
		beego.ControllerComments{
			"Put",
			`/:objectId`,
			[]string{"put"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:ObjectController"],
		beego.ControllerComments{
			"Delete",
			`/:objectId`,
			[]string{"delete"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:PingController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:PingController"],
		beego.ControllerComments{
			"Get",
			`/rest/ping.view`,
			[]string{"get"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"],
		beego.ControllerComments{
			"Post",
			`/`,
			[]string{"post"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"],
		beego.ControllerComments{
			"GetAll",
			`/`,
			[]string{"get"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"],
		beego.ControllerComments{
			"Get",
			`/:uid`,
			[]string{"get"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"],
		beego.ControllerComments{
			"Put",
			`/:uid`,
			[]string{"put"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"],
		beego.ControllerComments{
			"Delete",
			`/:uid`,
			[]string{"delete"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"],
		beego.ControllerComments{
			"Login",
			`/login`,
			[]string{"get"},
			nil})

	beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/deluan/gosonic/controllers:UserController"],
		beego.ControllerComments{
			"Logout",
			`/logout`,
			[]string{"get"},
			nil})

}
