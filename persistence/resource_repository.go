package persistence

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/deluan/rest"
	"github.com/google/uuid"
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

	// Get type of mappedModel (which is a *struct)
	rv := reflect.ValueOf(mappedModel)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	r.instanceType = rv.Type()
	r.sliceType = reflect.SliceOf(r.instanceType)
	return r
}

func (r *resourceRepository) EntityName() string {
	return r.instanceType.Name()
}

func (r *resourceRepository) newQuery(options ...rest.QueryOptions) orm.QuerySeter {
	qs := r.ormer.QueryTable(r.mappedModel)
	if len(options) > 0 {
		qs = r.addOptions(qs, options)
		qs = r.addFilters(qs, r.buildFilters(qs, options))
	}
	return qs
}

func (r *resourceRepository) NewInstance() interface{} {
	return reflect.New(r.instanceType).Interface()
}

func (r *resourceRepository) NewSlice() interface{} {
	slice := reflect.MakeSlice(r.sliceType, 0, 0)
	x := reflect.New(slice.Type())
	x.Elem().Set(slice)
	return x.Interface()
}

func (r *resourceRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	qs := r.newQuery(options...)
	dataSet := r.NewSlice()
	_, err := qs.All(dataSet)
	if err == orm.ErrNoRows {
		return dataSet, rest.ErrNotFound
	}
	return dataSet, err
}

func (r *resourceRepository) Count(options ...rest.QueryOptions) (int64, error) {
	qs := r.newQuery(options...)
	count, err := qs.Count()
	if err == orm.ErrNoRows {
		err = rest.ErrNotFound
	}
	return count, err
}

func (r *resourceRepository) Read(id string) (interface{}, error) {
	qs := r.newQuery().Filter("id", id)
	data := r.NewInstance()
	err := qs.One(data)
	if err == orm.ErrNoRows {
		return data, rest.ErrNotFound
	}
	return data, err
}

func setUUID(p interface{}) {
	f := reflect.ValueOf(p).Elem().FieldByName("ID")
	if f.Kind() == reflect.String {
		id, _ := uuid.NewRandom()
		f.SetString(id.String())
	}
}

func (r *resourceRepository) Save(p interface{}) (string, error) {
	setUUID(p)
	id, err := r.ormer.Insert(p)
	if err != nil {
		if err.Error() != "LastInsertId is not supported by this driver" {
			return "", err
		}
	}
	return strconv.FormatInt(id, 10), nil
}

func (r *resourceRepository) Update(p interface{}, cols ...string) error {
	count, err := r.ormer.Update(p, cols...)
	if err != nil {
		return err
	}
	if count == 0 {
		return rest.ErrNotFound
	}
	return err
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

func (r *resourceRepository) addFilters(qs orm.QuerySeter, conditions ...*orm.Condition) orm.QuerySeter {
	var cond *orm.Condition
	for _, c := range conditions {
		if c != nil {
			if cond == nil {
				cond = c
			} else {
				cond = cond.AndCond(c)
			}
		}
	}
	if cond != nil {
		return qs.SetCond(cond)
	}
	return qs
}

func unmarshalValue(val interface{}) string {
	switch v := val.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", val)
	}
}

func (r *resourceRepository) buildFilters(qs orm.QuerySeter, options []rest.QueryOptions) *orm.Condition {
	if len(options) == 0 {
		return nil
	}
	cond := orm.NewCondition()
	clauses := cond
	for f, v := range options[0].Filters {
		fn := strings.Replace(f, ".", "__", -1)
		s := unmarshalValue(v)

		if strings.HasSuffix(fn, "Id") || strings.HasSuffix(fn, "__id") {
			clauses = IdFilter(clauses, fn, s)
		} else {
			clauses = StartsWithFilter(clauses, fn, s)
		}
	}
	return clauses
}

func IdFilter(cond *orm.Condition, field, value string) *orm.Condition {
	field = strings.TrimSuffix(field, "Id") + "__id"
	return cond.And(field, value)
}

func StartsWithFilter(cond *orm.Condition, field, value string) *orm.Condition {
	return cond.And(field+"__istartswith", value)
}
