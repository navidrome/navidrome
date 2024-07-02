package main

import (
	"log"
	_ "net/http/pprof" //nolint:gosec

	"github.com/kardianos/service"
	"github.com/navidrome/navidrome/cmd"
)

var logger service.Logger

type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	cmd.Execute()
}
func (p *program) Stop(s service.Service) error {
	return nil
}

func main() {
	if service.Interactive() {
		cmd.Execute()
	} else {
		svcConfig := &service.Config{
			Name:        "Navidrome",
			DisplayName: "Navidrome",
			Description: "Your Personal Streaming Service",
		}

		prg := &program{}
		s, err := service.New(prg, svcConfig)
		if err != nil {
			log.Fatal(err)
		}
		logger, err = s.Logger(nil)
		if err != nil {
			log.Fatal(err)
		}
		err = s.Run()
		if err != nil {
			logErr := logger.Error(err)
			if logErr != nil {
				log.Println(logErr)
			}
		}
	}
}
