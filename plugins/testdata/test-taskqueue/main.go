// Test TaskQueue plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-taskqueue.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/taskworker"
)

func init() {
	taskworker.Register(&handler{})
}

type handler struct{}

func (h *handler) OnTaskExecute(req taskworker.TaskExecuteRequest) (taskworker.TaskExecuteResponse, error) {
	payload := string(req.Payload)
	if payload == "fail" {
		return taskworker.TaskExecuteResponse{Error: "task failed as instructed"}, nil
	}
	if payload == "fail-then-succeed" && req.Attempt < 2 {
		return taskworker.TaskExecuteResponse{Error: "transient failure"}, nil
	}
	return taskworker.TaskExecuteResponse{}, nil
}

// Test helper types
type TestInput struct {
	Operation string            `json:"operation"`
	QueueName string            `json:"queueName,omitempty"`
	Config    *host.QueueConfig `json:"config,omitempty"`
	Payload   []byte            `json:"payload,omitempty"`
	TaskID    string            `json:"taskId,omitempty"`
}

type TestOutput struct {
	TaskID string  `json:"taskId,omitempty"`
	Status string  `json:"status,omitempty"`
	Error  *string `json:"error,omitempty"`
}

//go:wasmexport nd_test_taskqueue
func ndTestTaskQueue() int32 {
	var input TestInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestOutput{Error: &errStr})
		return 0
	}

	switch input.Operation {
	case "create_queue":
		config := host.QueueConfig{}
		if input.Config != nil {
			config = *input.Config
		}
		err := host.TaskQueueCreateQueue(input.QueueName, config)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestOutput{})

	case "enqueue":
		taskID, err := host.TaskQueueEnqueue(input.QueueName, input.Payload)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestOutput{TaskID: taskID})

	case "get_task_status":
		status, err := host.TaskQueueGetTaskStatus(input.TaskID)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestOutput{Status: status})

	case "cancel_task":
		err := host.TaskQueueCancelTask(input.TaskID)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestOutput{})

	default:
		errStr := "unknown operation: " + input.Operation
		pdk.OutputJSON(TestOutput{Error: &errStr})
	}
	return 0
}

func main() {}
