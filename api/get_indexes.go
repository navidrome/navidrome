package api

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type GetIndexesController struct {
	BaseAPIController
	browser engine.Browser
}

func (c *GetIndexesController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.browser)
}

// TODO: Shortcuts amd validate musicFolder parameter
func (c *GetIndexesController) Get() {
	ifModifiedSince := c.ParamTime("ifModifiedSince")

	indexes, lastModified, err := c.browser.Indexes(ifModifiedSince)
	if err != nil {
		beego.Error("Error retrieving Indexes:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	res := responses.Indexes{
		IgnoredArticles: beego.AppConfig.String("ignoredArticles"),
		LastModified:    fmt.Sprint(utils.ToMillis(lastModified)),
	}

	res.Index = make([]responses.Index, len(*indexes))
	for i, idx := range *indexes {
		res.Index[i].Name = idx.Id
		res.Index[i].Artists = make([]responses.Artist, len(idx.Artists))
		for j, a := range idx.Artists {
			res.Index[i].Artists[j].Id = a.ArtistId
			res.Index[i].Artists[j].Name = a.Artist
		}
	}

	response := c.NewEmpty()
	response.Indexes = &res
	c.SendResponse(response)
}
