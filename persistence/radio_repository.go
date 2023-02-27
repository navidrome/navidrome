package persistence

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/deluan/rest"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type radioRepository struct {
	sqlRepository
	sqlRestful
	client httpDoer
}

func NewRadioRepository(ctx context.Context, o orm.QueryExecutor) model.RadioRepository {
	r := &radioRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "radio"
	r.filterMappings = map[string]filterFunc{
		"name": containsFilter,
	}
	r.client = &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	return r
}

func (r *radioRepository) isPermitted() bool {
	user := loggedUser(r.ctx)
	return user.IsAdmin
}

func (r *radioRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := r.newSelect(options...)
	return r.count(sql, options...)
}

func (r *radioRepository) Delete(id string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	return r.delete(Eq{"id": id})
}

func (r *radioRepository) Get(id string) (*model.Radio, error) {
	sel := r.newSelect().Where(Eq{"id": id}).Columns("*")
	res := model.Radio{}
	err := r.queryOne(sel, &res)

	if err != nil {
		return nil, err
	}

	if res.IsPlaylist {
		sel = Select("id", "name", "url").From("radio_link").Where(Eq{"radio_id": id})
		links := model.RadioLinks{}
		err = r.queryAll(sel, &links)

		if err != nil {
			return nil, err
		}

		res.Links = links
	}

	return &res, nil
}

func (r *radioRepository) GetAll(options ...model.QueryOptions) (model.Radios, error) {
	sel := r.newSelect(options...).Columns("*")
	res := model.Radios{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *radioRepository) Put(radio *model.Radio) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	log.Info(r.ctx, "PUT", "Radio", radio)

	var values map[string]interface{}

	radio.UpdatedAt = time.Now()

	if radio.ID == "" {
		radio.CreatedAt = time.Now()
		radio.ID = strings.ReplaceAll(uuid.NewString(), "-", "")
		values, _ = toSqlArgs(*radio)
	} else {
		values, _ = toSqlArgs(*radio)
		update := Update(r.tableName).Where(Eq{"id": radio.ID}).SetMap(values)

		count, err := r.executeSQL(update)

		if err != nil {
			return err
		} else if count > 0 {
			return r.refreshLinks(radio)
		}
	}

	values["created_at"] = time.Now()
	insert := Insert(r.tableName).SetMap(values)
	_, err := r.executeSQL(insert)

	if err != nil {
		return err
	}

	err = r.refreshLinks(radio)
	return err
}

func (r *radioRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *radioRepository) EntityName() string {
	return "radio"
}

func (r *radioRepository) NewInstance() interface{} {
	return &model.Radio{}
}

func (r *radioRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *radioRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

func (r *radioRepository) Save(entity interface{}) (string, error) {
	t := entity.(*model.Radio)
	if !r.isPermitted() {
		return "", rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return t.ID, err
}

func (r *radioRepository) Update(id string, entity interface{}, cols ...string) error {
	t := entity.(*model.Radio)
	t.ID = id
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

type Playlist int

const (
	M3U Playlist = iota
	PLS
)

const (
	MAX_PLS_BODY = 1024 * 1024 // 1 MiB
)

var (
	M3U_TYPES = map[string]Playlist{
		"application/mpegurl":                 M3U,
		"application/x-mpegurl":               M3U,
		"audio/mpegurl":                       M3U,
		"audio/x-mpegurl":                     M3U,
		"application/vnd.apple.mpegurl":       M3U,
		"application/vnd.apple.mpegurl.audio": M3U,
		"audio/x-scpls":                       PLS,
	}
	ErrLargePlaylistBody = errors.New("upstream playlist larger than 1 MB")
)

func (r *radioRepository) refreshLinks(radio *model.Radio) error {
	newReq, _ := http.NewRequestWithContext(r.ctx, "GET", radio.StreamUrl, nil)
	req, err := r.client.Do(newReq)

	if err != nil {
		return err
	}

	defer req.Body.Close()

	contentType := strings.TrimSpace(req.Header.Get("Content-Type"))
	pls, isPlaylist := M3U_TYPES[contentType]

	if !isPlaylist {
		return nil
	}

	if req.ContentLength > MAX_PLS_BODY {
		return ErrLargePlaylistBody
	}

	var links *model.RadioLinks

	if pls == M3U {
		links = r.parseM3u(radio.ID, req)
	} else {
		links, err = r.parsePls(radio.ID, req)
		if err != nil {
			return err
		}
	}

	del := Delete("radio_link").Where(Eq{"radio_id": radio.ID})
	_, err = r.executeSQL(del)
	if err != nil {
		return err
	}

	ins := Insert("radio_link").Columns("id", "name", "radio_id", "url")

	for _, link := range *links {
		ins = ins.Values(link.ID, link.Name, link.RadioId, link.Url)
	}

	_, err = r.executeSQL(ins)

	if err != nil {
		return err
	}

	now := time.Now()

	upd := Update(r.tableName).
		Set("is_playlist", true).
		Set("updated_at", now).
		Where(Eq{"id": radio.ID})

	_, err = r.executeSQL(upd)
	return err
}

func (r *radioRepository) parseM3u(id string, req *http.Response) *model.RadioLinks {
	scanner := bufio.NewScanner(req.Body)

	name := ""
	streamCount := 0

	var links model.RadioLinks

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF") {
			// Extended info can tell us the stream name. Otherwise, just
			// numbered 1...
			name = strings.Split(line, ",")[1]
		} else if !strings.HasPrefix(line, "#") {
			streamCount += 1

			if name == "" {
				name = strconv.Itoa(streamCount)
			}

			link := model.NewRadioLink(id, name, line)

			name = ""
			links = append(links, link)
		}
	}

	return &links
}

func (r *radioRepository) parsePls(id string, req *http.Response) (*model.RadioLinks, error) {
	scanner := bufio.NewScanner(req.Body)

	file := ""
	title := ""
	idx := 1

	var links model.RadioLinks

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		isFile := false
		isTitle := false

		if strings.HasPrefix(line, "File") {
			isFile = true
		} else if strings.HasPrefix(line, "Title") {
			isTitle = true
		}

		if !isFile && !isTitle {
			continue
		}

		data := strings.Split(line, "=")
		var idxString string

		if isFile {
			idxString = strings.TrimPrefix(data[0], "File")
		} else {
			idxString = strings.TrimPrefix(data[0], "Title")
		}

		curIdx, err := strconv.Atoi(idxString)
		if err != nil {
			return nil, err
		}

		if curIdx > idx {
			if file != "" {
				if title == "" {
					title = strconv.Itoa(idx)
				}

				link := model.RadioLink{
					ID:      strings.ReplaceAll(uuid.NewString(), "-", ""),
					Name:    title,
					RadioId: id,
					Url:     file,
				}
				links = append(links, link)
			}

			idx = curIdx

			if isTitle {
				file = ""
			} else {
				title = ""
			}
		}

		if isTitle {
			title = data[1]
		} else {
			file = data[1]
		}
	}

	if file != "" {
		if title == "" {
			title = strconv.Itoa(idx)
		}

		link := model.RadioLink{
			ID:      strings.ReplaceAll(uuid.NewString(), "-", ""),
			Name:    title,
			RadioId: id,
			Url:     file,
		}
		links = append(links, link)
	}

	return &links, nil
}

var _ model.RadioRepository = (*radioRepository)(nil)
var _ rest.Repository = (*radioRepository)(nil)
var _ rest.Persistable = (*radioRepository)(nil)
