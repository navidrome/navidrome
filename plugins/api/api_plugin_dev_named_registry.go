//go:build wasip1

package api

import (
	"context"
	"strings"

	"github.com/navidrome/navidrome/plugins/host/scheduler"
)

var callbacks = make(namedCallbacks)

// RegisterNamedSchedulerCallback registers a named scheduler callback. Named callbacks allow multiple callbacks to be registered
// within the same plugin, and for the schedules to be scoped to the named callback. If you only need a single callback, you can use
// the default (unnamed) callback registration function, RegisterSchedulerCallback.
// It returns a scheduler.SchedulerService that can be used to schedule jobs for the named callback.
//
// Notes:
//
// - You can't mix named and unnamed callbacks within the same plugin.
// - The name should be unique within the plugin, and it's recommended to use a short, descriptive name.
// - The name is case-sensitive.
func RegisterNamedSchedulerCallback(name string, cb SchedulerCallback) scheduler.SchedulerService {
	callbacks[name] = cb
	RegisterSchedulerCallback(&callbacks)
	return &namedSchedulerService{name: name, svc: scheduler.NewSchedulerService()}
}

const zwsp = string('\u200b')

// namedCallbacks is a map of named scheduler callbacks. The key is the name of the callback, and the value is the callback itself.
type namedCallbacks map[string]SchedulerCallback

func parseKey(key string) (string, string) {
	parts := strings.SplitN(key, zwsp, 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func (n *namedCallbacks) OnSchedulerCallback(ctx context.Context, req *SchedulerCallbackRequest) (*SchedulerCallbackResponse, error) {
	name, scheduleId := parseKey(req.ScheduleId)
	cb, exists := callbacks[name]
	if !exists {
		return nil, nil
	}
	req.ScheduleId = scheduleId
	return cb.OnSchedulerCallback(ctx, req)
}

// namedSchedulerService is a wrapper around the host scheduler service that prefixes the schedule IDs with the
// callback name. It is returned by RegisterNamedSchedulerCallback, and should be used by the plugin to schedule
// jobs for the named callback.
type namedSchedulerService struct {
	name string
	cb   SchedulerCallback
	svc  scheduler.SchedulerService
}

func (n *namedSchedulerService) makeKey(id string) string {
	return n.name + zwsp + id
}

func (n *namedSchedulerService) mapResponse(resp *scheduler.ScheduleResponse, err error) (*scheduler.ScheduleResponse, error) {
	if err != nil {
		return nil, err
	}
	_, resp.ScheduleId = parseKey(resp.ScheduleId)
	return resp, nil
}

func (n *namedSchedulerService) ScheduleOneTime(ctx context.Context, request *scheduler.ScheduleOneTimeRequest) (*scheduler.ScheduleResponse, error) {
	key := n.makeKey(request.ScheduleId)
	request.ScheduleId = key
	return n.mapResponse(n.svc.ScheduleOneTime(ctx, request))
}

func (n *namedSchedulerService) ScheduleRecurring(ctx context.Context, request *scheduler.ScheduleRecurringRequest) (*scheduler.ScheduleResponse, error) {
	key := n.makeKey(request.ScheduleId)
	request.ScheduleId = key
	return n.mapResponse(n.svc.ScheduleRecurring(ctx, request))
}

func (n *namedSchedulerService) CancelSchedule(ctx context.Context, request *scheduler.CancelRequest) (*scheduler.CancelResponse, error) {
	key := n.makeKey(request.ScheduleId)
	request.ScheduleId = key
	return n.svc.CancelSchedule(ctx, request)
}

func (n *namedSchedulerService) TimeNow(ctx context.Context, request *scheduler.TimeNowRequest) (*scheduler.TimeNowResponse, error) {
	return n.svc.TimeNow(ctx, request)
}
