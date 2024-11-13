package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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

	installUser      string
	workingDirectory string
)

func init() {
	svcCmd.AddCommand(buildInstallCmd())
	svcCmd.AddCommand(buildUninstallCmd())
	svcCmd.AddCommand(buildStartCmd())
	svcCmd.AddCommand(buildStopCmd())
	svcCmd.AddCommand(buildStatusCmd())
	svcCmd.AddCommand(buildExecuteCmd())
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
	done   chan struct{}
}

func (p *svcControl) Start(service.Service) error {
	p.done = make(chan struct{})
	p.ctx, p.cancel = context.WithCancel(context.Background())
	go func() {
		runNavidrome(p.ctx)
		close(p.done)
	}()
	return nil
}

func (p *svcControl) Stop(service.Service) error {
	log.Info("Stopping service")
	p.cancel()
	select {
	case <-p.done:
		log.Info("Service stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Error("Service did not stop in time. Killing it.")
	}
	return nil
}

var svcInstance = sync.OnceValue(func() service.Service {
	options := make(service.KeyValue)
	options["Restart"] = "on-failure"
	options["SuccessExitStatus"] = "1 2 8 SIGKILL"
	options["UserService"] = false
	options["LogDirectory"] = conf.Server.DataFolder
	options["SystemdScript"] = systemdScript
	if conf.Server.LogFile != "" {
		options["LogOutput"] = false
	} else {
		options["LogOutput"] = true
		options["LogDirectory"] = conf.Server.DataFolder
	}
	svcConfig := &service.Config{
		UserName:    installUser,
		Name:        "navidrome",
		DisplayName: "Navidrome",
		Description: "Your Personal Streaming Service",
		Dependencies: []string{
			"After=remote-fs.target network.target",
		},
		WorkingDirectory: executablePath(),
		Option:           options,
	}
	arguments := []string{"service", "execute"}
	if conf.Server.ConfigFile != "" {
		arguments = append(arguments, "-c", conf.Server.ConfigFile)
	}
	svcConfig.Arguments = arguments

	prg := &svcControl{}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	return svc
})

func runServiceCmd(cmd *cobra.Command, _ []string) {
	_ = cmd.Help()
}

func executablePath() string {
	if workingDirectory != "" {
		return workingDirectory
	}

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
		if conf.Server.LogFile != "" {
			println("  log file:          " + conf.Server.LogFile)
		} else {
			println("  logs folder:       " + conf.Server.DataFolder)
		}
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

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Navidrome service.",
		Run:   runInstallCmd,
	}
	cmd.Flags().StringVarP(&installUser, "user", "u", "", "user to run service")
	cmd.Flags().StringVarP(&workingDirectory, "working-directory", "w", "", "working directory of service")

	return cmd
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

func buildExecuteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "execute",
		Short: "Run navidrome as a service in the foreground (it is very unlikely you want to run this, you are better off running just navidrome)",
		Run: func(cmd *cobra.Command, args []string) {
			err := svcInstance().Run()
			if err != nil {
				log.Fatal(err)
			}
		},
	}
}

const systemdScript = `[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}} 
{{$dep}} {{end}}

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart={{.Path|cmdEscape}}{{range .Arguments}} {{.|cmd}}{{end}}
{{if .WorkingDirectory}}WorkingDirectory={{.WorkingDirectory|cmdEscape}}{{end}}
{{if .UserName}}User={{.UserName}}{{end}}
{{if .Restart}}Restart={{.Restart}}{{end}}
{{if .SuccessExitStatus}}SuccessExitStatus={{.SuccessExitStatus}}{{end}}
TimeoutStopSec=20
RestartSec=120
EnvironmentFile=-/etc/sysconfig/{{.Name}}

DevicePolicy=closed
NoNewPrivileges=yes
PrivateTmp=yes
ProtectControlGroups=yes
ProtectKernelModules=yes
ProtectKernelTunables=yes
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
RestrictNamespaces=yes
RestrictRealtime=yes
SystemCallFilter=~@clock @debug @module @mount @obsolete @reboot @setuid @swap
{{if .WorkingDirectory}}ReadWritePaths={{.WorkingDirectory|cmdEscape}}{{end}}
ProtectSystem=full

[Install]
WantedBy=multi-user.target
`
