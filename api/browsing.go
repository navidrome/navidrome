package api

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type BrowsingController struct {
	BaseAPIController
	browser engine.Browser
}

func (c *BrowsingController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.browser)
}

func (c *BrowsingController) GetMediaFolders() {
	mediaFolderList, _ := c.browser.MediaFolders()
	folders := make([]responses.MusicFolder, len(*mediaFolderList))
	for i, f := range *mediaFolderList {
		folders[i].Id = f.Id
		folders[i].Name = f.Name
	}
	response := c.NewEmpty()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	c.SendResponse(response)
}

// TODO: Shortcuts amd validate musicFolder parameter
func (c *BrowsingController) GetIndexes() {
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

func (c *BrowsingController) GetDirectory() {
	id := c.RequiredParamString("id", "id parameter required")

	response := c.NewEmpty()

	dir, err := c.browser.Directory(id)
	switch {
	case err == engine.ErrDataNotFound:
		beego.Error("Requested Id", id, "not found:", err)
		c.SendError(responses.ERROR_DATA_NOT_FOUND, "Directory not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	response.Directory = c.buildDirectory(dir)

	c.SendResponse(response)
}

func (c *BrowsingController) buildDirectory(d *engine.DirectoryInfo) *responses.Directory {
	dir := &responses.Directory{Id: d.Id, Name: d.Name}

	dir.Child = make([]responses.Child, len(d.Entries))
	for i, entry := range d.Entries {
		dir.Child[i] = c.ToChild(entry)
	}
	return dir
}
