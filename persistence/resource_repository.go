package persistence

import (
	"reflect"
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/deluan/rest"
)

type resourceRepository struct {
	model.ResourceRepository
	model        interface{}
	mappedModel  interface{}
	ormer        orm.Ormer
	instanceType reflect.Type
	sliceType    reflect.Type
}

func NewResource(o orm.Ormer, model interface{}, mappedModel interface{}) model.ResourceRepository {
	r := &resourceRepository{model: model, mappedModel: mappedModel, ormer: o}
	r.instanceType = reflect.TypeOf(mappedModel)
	r.sliceType = reflect.SliceOf(r.instanceType)
	return r
}

func (r *resourceRepository) newQuery() orm.QuerySeter {
	return r.ormer.QueryTable(r.mappedModel)
}

func (r *resourceRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	qs := r.newQuery()
	qs = r.addOptions(qs, options)
	//qs = r.addFilters(qs, r.buildFilters(qs, options), r.getRestriction())
	dataSet := r.NewSlice()
	_, err := qs.All(dataSet)
	if err == orm.ErrNoRows {
		return dataSet, rest.ErrNotFound
	}
	return dataSet, err
}

func (r *resourceRepository) Count(options ...rest.QueryOptions) (int64, error) {
	qs := r.newQuery()
	//qs = r.addFilters(qs, r.buildFilters(qs, options), r.getRestriction())
	count, err := qs.Count()
	if err == orm.ErrNoRows {
		err = rest.ErrNotFound
	}
	return count, err
}

func (r *resourceRepository) NewSlice() interface{} {
	slice := reflect.MakeSlice(r.sliceType, 0, 0)
	x := reflect.New(slice.Type())
	x.Elem().Set(slice)
	return x.Interface()
}

func (r *resourceRepository) addOptions(qs orm.QuerySeter, options []rest.QueryOptions) orm.QuerySeter {
	if len(options) == 0 {
		return qs
	}
	opt := options[0]
	sort := strings.Split(opt.Sort, ",")
	reverse := strings.ToLower(opt.Order) == "desc"
	for i, s := range sort {
		s = strings.TrimSpace(s)
		if reverse {
			if s[0] == '-' {
				s = strings.TrimPrefix(s, "-")
			} else {
				s = "-" + s
			}
		}
		sort[i] = strings.Replace(s, ".", "__", -1)
	}
	if opt.Sort != "" {
		qs = qs.OrderBy(sort...)
	}
	if opt.Max > 0 {
		qs = qs.Limit(opt.Max)
	}
	if opt.Offset > 0 {
		qs = qs.Offset(opt.Offset)
	}
	return qs
}
