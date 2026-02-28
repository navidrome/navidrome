// Test TaskQueue plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-taskqueue.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"fmt"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/taskworker"
)

func init() {
	taskworker.Register(&handler{})
}

type handler struct{}

func (h *handler) OnTaskExecute(req taskworker.TaskExecuteRequest) (string, error) {
	payload := string(req.Payload)
	if payload == "fail" {
		return "", fmt.Errorf("task failed as instructed")
	}
	if payload == "fail-then-succeed" && req.Attempt < 2 {
		return "", fmt.Errorf("transient failure")
	}
	return "completed successfully", nil
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
	TaskID  string  `json:"taskId,omitempty"`
	Status  string  `json:"status,omitempty"`
	Message string  `json:"message,omitempty"`
	Attempt int32   `json:"attempt,omitempty"`
	Error   *string `json:"error,omitempty"`
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
		err := host.TaskCreateQueue(input.QueueName, config)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestOutput{})

	case "enqueue":
		taskID, err := host.TaskEnqueue(input.QueueName, input.Payload)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestOutput{TaskID: taskID})

	case "get_task_status":
		info, err := host.TaskGet(input.TaskID)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(TestOutput{Error: &errStr})
			return 0
		}
		pdk.OutputJSON(TestOutput{Status: info.Status, Message: info.Message, Attempt: info.Attempt})

	case "cancel_task":
		err := host.TaskCancel(input.TaskID)
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
