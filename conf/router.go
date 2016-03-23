package conf

import (
	"github.com/deluan/gosonic/api"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/deluan/gosonic/controllers"
)

func init() {
	mapEndpoints()
	mapControllers()
	mapFilters()
}

func mapEndpoints() {
	ns := beego.NewNamespace("/rest",
		beego.NSRouter("/ping.view", &api.SystemController{}, "*:Ping"),
		beego.NSRouter("/getLicense.view", &api.SystemController{}, "*:GetLicense"),

		beego.NSRouter("/getMusicFolders.view", &api.BrowsingController{}, "*:GetMediaFolders"),
		beego.NSRouter("/getIndexes.view", &api.BrowsingController{}, "*:GetIndexes"),
		beego.NSRouter("/getMusicDirectory.view", &api.BrowsingController{}, "*:GetDirectory"),

		beego.NSRouter("/search2.view", &api.SearchingController{}, "*:Search2"),

		beego.NSRouter("/getCoverArt.view", &api.MediaRetrievalController{}, "*:GetCover"),
		beego.NSRouter("/getAvatar.view", &api.MediaRetrievalController{}, "*:GetAvatar"),
		beego.NSRouter("/stream.view", &api.StreamController{}, "*:Stream"),
		beego.NSRouter("/download.view", &api.StreamController{}, "*:Download"),

		beego.NSRouter("/scrobble.view", &api.MediaAnnotationController{}, "*:Scrobble"),
		beego.NSRouter("/star.view", &api.MediaAnnotationController{}, "*:Star"),
		beego.NSRouter("/unstar.view", &api.MediaAnnotationController{}, "*:Unstar"),

		beego.NSRouter("/getAlbumList.view", &api.AlbumListController{}, "*:GetAlbumList"),
		beego.NSRouter("/getStarred.view", &api.AlbumListController{}, "*:GetStarred"),
		beego.NSRouter("/getNowPlaying.view", &api.AlbumListController{}, "*:GetNowPlaying"),

		beego.NSRouter("/getPlaylists.view", &api.PlaylistsController{}, "*:GetAll"),
		beego.NSRouter("/getPlaylist.view", &api.PlaylistsController{}, "*:Get"),

		beego.NSRouter("/getUser.view", &api.UsersController{}, "*:GetUser"),
	)
	beego.AddNamespace(ns)

}

func mapControllers() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/sync", &controllers.SyncController{})

	beego.ErrorController(&controllers.MainController{})
}

func mapFilters() {
	var ValidateRequest = func(ctx *context.Context) {
		c := api.BaseAPIController{}
		c.Ctx = ctx
		c.Data = make(map[interface{}]interface{})
		api.Validate(c)
	}

	beego.InsertFilter("/rest/*", beego.BeforeRouter, ValidateRequest)
}
