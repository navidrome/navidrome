package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/oklog/run"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	noBanner bool

	rootCmd = &cobra.Command{
		Use:   "navidrome",
		Short: "Navidrome is a self-hosted music server and streamer",
		Long: `Navidrome is a self-hosted music server and streamer.
Complete documentation is available at https://www.navidrome.org/docs`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			preRun()
		},
		Run: func(cmd *cobra.Command, args []string) {
			runNavidrome()
		},
		Version: consts.Version(),
	}
)

func Execute() {
	rootCmd.SetVersionTemplate(`{{println .Version}}`)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func preRun() {
	if !noBanner {
		println(consts.Banner())
	}
	conf.Load()
}

func runNavidrome() {
	db.EnsureLatestVersion()

	var g run.Group

	g.Add(startServer())
	g.Add(startSignaler())
	g.Add(startScheduler())

	schedule := conf.Server.ScanSchedule
	if schedule != "" {
		go schedulePeriodicScan(schedule)
	} else {
		log.Warn("Periodic scan is DISABLED", "schedule", schedule)
	}

	if err := g.Run(); err != nil {
		log.Error("Fatal error in Navidrome. Aborting", err)
	}
}

func startServer() (func() error, func(err error)) {
	return func() error {
			a := CreateServer(conf.Server.MusicFolder)
			a.MountRouter(consts.URLPathSubsonicAPI, CreateSubsonicAPIRouter())
			a.MountRouter(consts.URLPathUI, CreateAppRouter())
			return a.Run(fmt.Sprintf("%s:%d", conf.Server.Address, conf.Server.Port))
		}, func(err error) {
			if err != nil {
				log.Error("Shutting down Server due to error", err)
			} else {
				log.Info("Shutting down Server")
			}
		}
}

var sigChan = make(chan os.Signal, 1)

func startSignaler() (func() error, func(err error)) {
	scanner := GetScanner()
	ctx, cancel := context.WithCancel(context.Background())

	return func() error {
			for {
				select {
				case sig := <-sigChan:
					log.Info(ctx, "Received signal, triggering a new scan", "signal", sig)
					start := time.Now()
					err := scanner.RescanAll(ctx, false)
					if err != nil {
						log.Error(ctx, "Error scanning", err)
					}
					log.Info(ctx, "Triggered scan complete", "elapsed", time.Since(start).Round(100*time.Millisecond))
				case <-ctx.Done():
					break
				}
			}
		}, func(err error) {
			cancel()
			if err != nil {
				log.Error("Shutting down Signaler due to error", err)
			} else {
				log.Info("Shutting down Signaler")
			}
		}
}

func schedulePeriodicScan(schedule string) {
	scanner := GetScanner()
	scheduler := GetScheduler()

	log.Info("Scheduling periodic scan", "schedule", schedule)
	err := scheduler.Add(schedule, func() {
		_ = scanner.RescanAll(context.Background(), false)
	})
	if err != nil {
		log.Error("Error scheduling periodic scan", err)
	}

	time.Sleep(2 * time.Second) // Wait 2 seconds before the initial scan
	log.Info("Executing initial scan")
	if err := scanner.RescanAll(context.Background(), false); err != nil {
		log.Error("Error executing initial scan", err)
	}
}

func startScheduler() (func() error, func(err error)) {
	log.Info("Starting scheduler")
	scheduler := GetScheduler()
	ctx, cancel := context.WithCancel(context.Background())

	return func() error {
			scheduler.Run(ctx)

			return nil
		}, func(err error) {
			cancel()
			if err != nil {
				log.Error("Shutting down Scheduler due to error", err)
			} else {
				log.Info("Shutting down Scheduler")
			}
		}
}

// TODO: Implement some struct tags to map flags to viper
func init() {
	cobra.OnInitialize(func() {
		conf.InitConfig(cfgFile)
	})

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "configfile", "c", "", `config file (default "./navidrome.toml")`)
	rootCmd.PersistentFlags().BoolVarP(&noBanner, "nobanner", "n", false, `don't show banner`)
	rootCmd.PersistentFlags().String("musicfolder", viper.GetString("musicfolder"), "folder where your music is stored")
	rootCmd.PersistentFlags().String("datafolder", viper.GetString("datafolder"), "folder to store application data (DB, cache...), needs write access")
	rootCmd.PersistentFlags().StringP("loglevel", "l", viper.GetString("loglevel"), "log level, possible values: error, info, debug, trace")

	_ = viper.BindPFlag("musicfolder", rootCmd.PersistentFlags().Lookup("musicfolder"))
	_ = viper.BindPFlag("datafolder", rootCmd.PersistentFlags().Lookup("datafolder"))
	_ = viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))

	rootCmd.Flags().StringP("address", "a", viper.GetString("address"), "IP address to bind")
	rootCmd.Flags().IntP("port", "p", viper.GetInt("port"), "HTTP port Navidrome will use")
	rootCmd.Flags().Duration("sessiontimeout", viper.GetDuration("sessiontimeout"), "how long Navidrome will wait before closing web ui idle sessions")
	rootCmd.Flags().Duration("scaninterval", viper.GetDuration("scaninterval"), "how frequently to scan for changes in your music library")
	rootCmd.Flags().String("baseurl", viper.GetString("baseurl"), "base URL (only the path part) to configure Navidrome behind a proxy (ex: /music)")
	rootCmd.Flags().String("uiloginbackgroundurl", viper.GetString("uiloginbackgroundurl"), "URL to a backaground image used in the Login page")
	rootCmd.Flags().Bool("enabletranscodingconfig", viper.GetBool("enabletranscodingconfig"), "enables transcoding configuration in the UI")
	rootCmd.Flags().String("transcodingcachesize", viper.GetString("transcodingcachesize"), "size of transcoding cache")
	rootCmd.Flags().String("imagecachesize", viper.GetString("imagecachesize"), "size of image (art work) cache. set to 0 to disable cache")
	rootCmd.Flags().Bool("autoimportplaylists", viper.GetBool("autoimportplaylists"), "enable/disable .m3u playlist auto-import`")

	_ = viper.BindPFlag("address", rootCmd.Flags().Lookup("address"))
	_ = viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("sessiontimeout", rootCmd.Flags().Lookup("sessiontimeout"))
	_ = viper.BindPFlag("scaninterval", rootCmd.Flags().Lookup("scaninterval"))
	_ = viper.BindPFlag("baseurl", rootCmd.Flags().Lookup("baseurl"))
	_ = viper.BindPFlag("uiloginbackgroundurl", rootCmd.Flags().Lookup("uiloginbackgroundurl"))
	_ = viper.BindPFlag("enabletranscodingconfig", rootCmd.Flags().Lookup("enabletranscodingconfig"))
	_ = viper.BindPFlag("transcodingcachesize", rootCmd.Flags().Lookup("transcodingcachesize"))
	_ = viper.BindPFlag("imagecachesize", rootCmd.Flags().Lookup("imagecachesize"))
}
