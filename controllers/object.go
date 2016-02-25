package controllers

import (
	"encoding/json"
	"github.com/deluan/gosonic/models"

	"github.com/astaxie/beego"
)

// Operations about object
type ObjectController struct {
	beego.Controller
}

// @Title create
// @Description create object
// @Param	body		body 	models.Object	true		"The object content"
// @Success 200 {string} models.Object.Id
// @Failure 403 body is empty
// @router / [post]
func (o *ObjectController) Post() {
	var ob models.Object
	json.Unmarshal(o.Ctx.Input.RequestBody, &ob)
	objectid := models.AddOne(ob)
	o.Data["json"] = map[string]string{"ObjectId": objectid}
	o.ServeJSON()
}

// @Title Get
// @Description find object by objectid
// @Param	objectId		path 	string	true		"the objectid you want to get"
// @Success 200 {object} models.Object
// @Failure 403 :objectId is empty
// @router /:objectId [get]
func (o *ObjectController) Get() {
	objectId := o.Ctx.Input.Param(":objectId")
	if objectId != "" {
		ob, err := models.GetOne(objectId)
		if err != nil {
			o.Data["json"] = err.Error()
		} else {
			o.Data["json"] = ob
		}
	}
	o.ServeJSON()
}

// @Title GetAll
// @Description get all objects
// @Success 200 {object} models.Object
// @Failure 403 :objectId is empty
// @router / [get]
func (o *ObjectController) GetAll() {
	obs := models.GetAll()
	o.Data["json"] = obs
	o.ServeJSON()
}

// @Title update
// @Description update the object
// @Param	objectId		path 	string	true		"The objectid you want to update"
// @Param	body		body 	models.Object	true		"The body"
// @Success 200 {object} models.Object
// @Failure 403 :objectId is empty
// @router /:objectId [put]
func (o *ObjectController) Put() {
	objectId := o.Ctx.Input.Param(":objectId")
	var ob models.Object
	json.Unmarshal(o.Ctx.Input.RequestBody, &ob)

	err := models.Update(objectId, ob.Score)
	if err != nil {
		o.Data["json"] = err.Error()
	} else {
		o.Data["json"] = "update success!"
	}
	o.ServeJSON()
}

// @Title delete
// @Description delete the object
// @Param	objectId		path 	string	true		"The objectId you want to delete"
// @Success 200 {string} delete success!
// @Failure 403 objectId is empty
// @router /:objectId [delete]
func (o *ObjectController) Delete() {
	objectId := o.Ctx.Input.Param(":objectId")
	models.Delete(objectId)
	o.Data["json"] = "delete success!"
	o.ServeJSON()
}
