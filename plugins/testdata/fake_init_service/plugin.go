//go:build wasip1

package main

import (
	"context"
	"errors"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
)

type initServicePlugin struct{}

func (p *initServicePlugin) OnInit(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
	log.Printf("OnInit called with %v", req)

	// Check for specific error conditions in the config
	if req.Config != nil {
		if errorType, exists := req.Config["returnError"]; exists {
			switch errorType {
			case "go_error":
				return nil, errors.New("initialization failed with Go error")
			case "response_error":
				return &api.InitResponse{
					Error: "initialization failed with response error",
				}, nil
			}
		}
	}

	// Default: successful initialization
	return &api.InitResponse{}, nil
}

// Required by Go WASI build
func main() {}

// Register the LifecycleManagement implementation
func init() {
	api.RegisterLifecycleManagement(&initServicePlugin{})
}
