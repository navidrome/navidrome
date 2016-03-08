package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type GetMusicDirectoryController struct {
	BaseAPIController
	browser engine.Browser
}

func (c *GetMusicDirectoryController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.browser)
}

func (c *GetMusicDirectoryController) Get() {
	id := c.RequiredParamString("id", "id parameter required")

	response := c.NewEmpty()

	dir, err := c.browser.Directory(id)
	switch {
	case err == engine.DataNotFound:
		beego.Error(err, "Id:", id)
		c.SendError(responses.ERROR_DATA_NOT_FOUND, "Directory not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	response.Directory = c.buildDirectory(dir)

	c.SendResponse(response)
}

func (c *GetMusicDirectoryController) buildDirectory(d *engine.DirectoryInfo) *responses.Directory {
	dir := &responses.Directory{Id: d.Id, Name: d.Name}

	dir.Child = make([]responses.Child, len(d.Children))
	for i, child := range d.Children {
		dir.Child[i].Id = child.Id
		dir.Child[i].Title = child.Title
		dir.Child[i].IsDir = child.IsDir
		dir.Child[i].Parent = child.Parent
		dir.Child[i].Album = child.Album
		dir.Child[i].Year = child.Year
		dir.Child[i].Artist = child.Artist
		dir.Child[i].Genre = child.Genre
		dir.Child[i].CoverArt = child.CoverArt
		dir.Child[i].Track = child.Track
		dir.Child[i].Duration = child.Duration
		dir.Child[i].Size = child.Size
		dir.Child[i].Suffix = child.Suffix
		dir.Child[i].BitRate = child.BitRate
		dir.Child[i].ContentType = child.ContentType
		if !child.Starred.IsZero() {
			dir.Child[i].Starred = &child.Starred
		}
	}
	return dir
}
