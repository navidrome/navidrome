package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/consts"
	"strconv"
	"github.com/deluan/gosonic/domain"
)

type GetIndexesController struct {
	beego.Controller
	repo       domain.ArtistIndexRepository
	properties domain.PropertyRepository
}

func (c *GetIndexesController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.repo)
	inject.ExtractAssignable(utils.Graph, &c.properties)
}

// TODO: Shortcuts amd validate musicFolder parameter
func (c *GetIndexesController) Get() {
	var err error

	ifModifiedSince := c.Input().Get("ifModifiedSince")
	if ifModifiedSince == "" {
		ifModifiedSince = "0"
	}

	res := responses.ArtistIndex{}
	res.IgnoredArticles = beego.AppConfig.String("ignoredArticles")

	res.LastModified, err = c.properties.DefaultGet(consts.LastScan, "-1")
	if err != nil {
		beego.Error("Error retrieving LastScan property:", err)
		c.CustomAbort(200, string(responses.NewError(responses.ERROR_GENERIC, "Internal Error")))
	}

	i, _ := strconv.Atoi(ifModifiedSince)
	l, _ := strconv.Atoi(res.LastModified)

	if (l > i) {
		indexes, err := c.repo.GetAll()
		if err != nil {
			beego.Error("Error retrieving Indexes:", err)
			c.CustomAbort(200, string(responses.NewError(responses.ERROR_GENERIC, "Internal Error")))
		}

		res.Index = make([]responses.IdxIndex, len(indexes))
		for i, idx := range indexes {
			res.Index[i].Name = idx.Id
			res.Index[i].Artists = make([]responses.IdxArtist, len(idx.Artists))
			for j, a := range idx.Artists {
				res.Index[i].Artists[j].Id = a.ArtistId
				res.Index[i].Artists[j].Name = a.Artist
			}
		}

	}

	response := responses.NewEmpty()
	response.ArtistIndex = res
	c.Ctx.Output.Body(responses.ToXML(response))
}
