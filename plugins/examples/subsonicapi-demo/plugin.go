//go:build wasip1

package main

import (
	"context"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/subsonicapi"
)

// SubsonicAPIService instance for making API calls
var subsonicService = subsonicapi.NewSubsonicAPIService()

// SubsonicAPIDemoPlugin implements LifecycleManagement interface
type SubsonicAPIDemoPlugin struct{}

// OnInit is called when the plugin is loaded
func (SubsonicAPIDemoPlugin) OnInit(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
	log.Printf("SubsonicAPI Demo Plugin initializing...")

	// Example: Call the ping endpoint to check if the server is alive
	response, err := subsonicService.Call(ctx, &subsonicapi.CallRequest{
		Url: "/rest/ping?u=admin",
	})

	if err != nil {
		log.Printf("SubsonicAPI call failed: %v", err)
		return &api.InitResponse{Error: err.Error()}, nil
	}

	if response.Error != "" {
		log.Printf("SubsonicAPI returned error: %s", response.Error)
		return &api.InitResponse{Error: response.Error}, nil
	}

	log.Printf("SubsonicAPI ping response: %s", response.Json)

	// Example: Get server info
	infoResponse, err := subsonicService.Call(ctx, &subsonicapi.CallRequest{
		Url: "/rest/getLicense?u=admin",
	})

	if err != nil {
		log.Printf("SubsonicAPI getLicense call failed: %v", err)
		return &api.InitResponse{Error: err.Error()}, nil
	}

	if infoResponse.Error != "" {
		log.Printf("SubsonicAPI getLicense returned error: %s", infoResponse.Error)
		return &api.InitResponse{Error: infoResponse.Error}, nil
	}

	log.Printf("SubsonicAPI license info: %s", infoResponse.Json)

	return &api.InitResponse{}, nil
}

func main() {}

func init() {
	// Configure logging: No timestamps, no source file/line
	log.SetFlags(0)
	log.SetPrefix("[Subsonic Plugin] ")

	api.RegisterLifecycleManagement(&SubsonicAPIDemoPlugin{})
}
