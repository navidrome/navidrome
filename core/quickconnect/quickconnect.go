// Package quickconnect implements Jellyfin-style Quick Connect: a new client shows a short code, and
// an already signed-in user approves it, so the client can sign in without a password.
package quickconnect

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/random"
	"github.com/navidrome/navidrome/utils/singleton"
)

const timeout = 10 * time.Minute

// Initiate is unauthenticated, so the number of live requests must be bounded.
const maxPending = 1000

var (
	ErrAlreadyAuthorized = errors.New("quick connect request already authorized")
	ErrTooManyRequests   = errors.New("too many pending quick connect requests")
)

type Device struct {
	ID         string
	Name       string
	App        string
	AppVersion string
}

type Request struct {
	Device    Device
	Secret    string
	Code      string
	DateAdded time.Time
	UserID    string
}

func (r Request) Authorized() bool {
	return r.UserID != ""
}

type QuickConnect interface {
	Initiate(device Device) (Request, error)
	Status(secret string) (Request, error)
	// Lookup returns a request still waiting for approval.
	Lookup(code string) (Request, error)
	Authorize(code, userID string) (Request, error)
	// Redeem returns the id of the user who approved the request. It succeeds only once per secret.
	Redeem(secret string) (string, error)
}

type entry struct {
	Request
	expiresAt time.Time
}

type quickConnect struct {
	mu       sync.Mutex
	bySecret map[string]*entry
	byCode   map[string]*entry
}

// GetInstance returns the shared store: the Jellyfin and native API routers must see the same requests.
func GetInstance() QuickConnect {
	return singleton.GetInstance(newStore)
}

func New() QuickConnect {
	return newStore()
}

func newStore() *quickConnect {
	return &quickConnect{
		bySecret: map[string]*entry{},
		byCode:   map[string]*entry{},
	}
}

// Enabled reports whether users can approve codes from the web UI, which needs the Jellyfin API.
func Enabled() bool {
	return conf.Server.Jellyfin.Enabled && conf.Server.Jellyfin.QuickConnect
}

func (qc *quickConnect) Initiate(device Device) (Request, error) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.expire()
	if len(qc.bySecret) >= maxPending {
		return Request{}, ErrTooManyRequests
	}
	// The fields may be substrings of a much larger header; copy them so the header isn't retained.
	device = Device{ID: strings.Clone(device.ID), Name: strings.Clone(device.Name),
		App: strings.Clone(device.App), AppVersion: strings.Clone(device.AppVersion)}
	now := time.Now()
	e := &entry{
		Request:   Request{Device: device, Secret: newSecret(), Code: qc.newCode(), DateAdded: now},
		expiresAt: now.Add(timeout),
	}
	qc.bySecret[e.Secret] = e
	qc.byCode[e.Code] = e
	return e.Request, nil
}

func (qc *quickConnect) Status(secret string) (Request, error) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	e, err := qc.find(qc.bySecret, secret)
	if err != nil {
		return Request{}, err
	}
	return e.Request, nil
}

func (qc *quickConnect) Lookup(code string) (Request, error) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	e, err := qc.findPending(code)
	if err != nil {
		return Request{}, err
	}
	return e.Request, nil
}

func (qc *quickConnect) Authorize(code, userID string) (Request, error) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	e, err := qc.findPending(code)
	if err != nil {
		return Request{}, err
	}
	e.UserID = userID
	e.expiresAt = time.Now().Add(timeout)
	return e.Request, nil
}

func (qc *quickConnect) Redeem(secret string) (string, error) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	e, err := qc.find(qc.bySecret, secret)
	if err != nil || !e.Authorized() {
		return "", model.ErrNotFound
	}
	qc.remove(e)
	return e.UserID, nil
}

func (qc *quickConnect) find(index map[string]*entry, key string) (*entry, error) {
	qc.expire()
	e, ok := index[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return e, nil
}

func (qc *quickConnect) findPending(code string) (*entry, error) {
	e, err := qc.find(qc.byCode, normalizeCode(code))
	if err == nil && e.Authorized() {
		return nil, ErrAlreadyAuthorized
	}
	return e, err
}

func (qc *quickConnect) expire() {
	now := time.Now()
	for _, e := range qc.bySecret {
		if !now.Before(e.expiresAt) {
			qc.remove(e)
		}
	}
}

func (qc *quickConnect) remove(e *entry) {
	delete(qc.bySecret, e.Secret)
	delete(qc.byCode, e.Code)
}

func (qc *quickConnect) newCode() string {
	for {
		code := fmt.Sprintf("%06d", random.Int64N(900000)+100000)
		if _, taken := qc.byCode[code]; !taken {
			return code
		}
	}
}

func newSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Users often type the code in groups ("123 456").
func normalizeCode(code string) string {
	return strings.Join(strings.Fields(code), "")
}
