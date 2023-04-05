package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kardianos/service"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/spf13/cobra"
)

var (
	svcStatusLabels = map[service.Status]string{
		service.StatusUnknown: "Unknown",
		service.StatusStopped: "Stopped",
		service.StatusRunning: "Running",
	}
)

func init() {
	svcCmd.AddCommand(buildInstallCmd())
	svcCmd.AddCommand(buildUninstallCmd())
	svcCmd.AddCommand(buildStartCmd())
	svcCmd.AddCommand(buildStopCmd())
	svcCmd.AddCommand(buildStatusCmd())
	rootCmd.AddCommand(svcCmd)
}

var svcCmd = &cobra.Command{
	Use:     "service",
	Aliases: []string{"svc"},
	Short:   "Manage Navidrome as a service",
	Long:    fmt.Sprintf("Manage Navidrome as a service, using the OS service manager (%s)", service.Platform()),
	Run:     runServiceCmd,
}

type svcControl struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (p *svcControl) Start(_ service.Service) error {
	p.ctx, p.cancel = context.WithCancel(context.Background())
	go p.run()
	return nil
}

func (p *svcControl) run() {
	runNavidrome(p.ctx)
}

func (p *svcControl) Stop(_ service.Service) error {
	log.Info("Stopping service")
	p.cancel()
	return nil
}

var (
	svc     service.Service
	svcOnce = sync.Once{}
)

func svcInstance() service.Service {
	svcOnce.Do(func() {
		options := make(service.KeyValue)
		options["Restart"] = "on-success"
		options["SuccessExitStatus"] = "1 2 8 SIGKILL"
		options["UserService"] = true
		options["LogDirectory"] = conf.Server.DataFolder
		svcConfig := &service.Config{
			Name:        "Navidrome",
			DisplayName: "Navidrome",
			Description: "Navidrome is a self-hosted music server and streamer",
			Dependencies: []string{
				"Requires=network.target",
				"After=network-online.target syslog.target"},
			WorkingDirectory: executablePath(),
			Option:           options,
		}
		if conf.Server.ConfigFile != "" {
			svcConfig.Arguments = []string{"-c", conf.Server.ConfigFile}
		}
		prg := &svcControl{}
		var err error
		svc, err = service.New(prg, svcConfig)
		if err != nil {
			log.Fatal(err)
		}
	})
	return svc
}

func runServiceCmd(cmd *cobra.Command, _ []string) {
	_ = cmd.Help()
}

func executablePath() string {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Dir(ex)
}

func buildInstallCmd() *cobra.Command {
	runInstallCmd := func(_ *cobra.Command, _ []string) {
		var err error
		println("Installing service with:")
		println("  working directory: " + executablePath())
		println("  music folder:      " + conf.Server.MusicFolder)
		println("  data folder:       " + conf.Server.DataFolder)
		if cfgFile != "" {
			conf.Server.ConfigFile, err = filepath.Abs(cfgFile)
			if err != nil {
				log.Fatal(err)
			}
			println("  config file:       " + conf.Server.ConfigFile)
		}
		err = svcInstance().Install()
		if err != nil {
			log.Fatal(err)
		}
		println("Service installed. Use 'navidrome svc start' to start it.")
	}

	return &cobra.Command{
		Use:   "install",
		Short: "Install Navidrome service.",
		Run:   runInstallCmd,
	}
}

func buildUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Navidrome service. Does not delete the music or data folders",
		Run: func(cmd *cobra.Command, args []string) {
			err := svcInstance().Uninstall()
			if err != nil {
				log.Fatal(err)
			}
			println("Service uninstalled. Music and data folders are still intact.")
		},
	}
}

func buildStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start Navidrome service",
		Run: func(cmd *cobra.Command, args []string) {
			err := svcInstance().Start()
			if err != nil {
				log.Fatal(err)
			}
			println("Service started. Use 'navidrome svc status' to check its status.")
		},
	}
}

func buildStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop Navidrome service",
		Run: func(cmd *cobra.Command, args []string) {
			err := svcInstance().Stop()
			if err != nil {
				log.Fatal(err)
			}
			println("Service stopped. Use 'navidrome svc status' to check its status.")
		},
	}
}

func buildStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Navidrome service status",
		Run: func(cmd *cobra.Command, args []string) {
			status, err := svcInstance().Status()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Navidrome is %s.\n", svcStatusLabels[status])
		},
	}
}
