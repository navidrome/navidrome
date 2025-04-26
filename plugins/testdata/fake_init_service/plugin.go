//go:build wasip1

package main

import (
	"context"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
)

type initServicePlugin struct{}

func (p *initServicePlugin) OnInit(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
	log.Printf("OnInit called with %v", req)
	return &api.InitResponse{}, nil
}

func main() {
	api.RegisterInitService(&initServicePlugin{})
}
