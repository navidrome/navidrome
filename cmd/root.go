package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/scheduler"
	"github.com/navidrome/navidrome/server/backgrounds"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var interrupted = errors.New("service was interrupted")

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
		Version: consts.Version,
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
		println(resources.Banner())
	}
	conf.Load()
}

func runNavidrome() {
	db.Init()
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Error closing DB", err)
		}
		log.Info("Navidrome stopped, bye.")
	}()

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(startServer(ctx))
	g.Go(startSignaler(ctx))
	g.Go(startScheduler(ctx))
	g.Go(schedulePeriodicScan(ctx))

	if conf.Server.Jukebox.Enabled {
		g.Go(startPlaybackServer(ctx))
	}

	if err := g.Wait(); err != nil && !errors.Is(err, interrupted) {
		log.Error("Fatal error in Navidrome. Aborting", err)
	}
}

func startServer(ctx context.Context) func() error {
	return func() error {
		a := CreateServer(conf.Server.MusicFolder)
		a.MountRouter("Native API", consts.URLPathNativeAPI, CreateNativeAPIRouter())
		a.MountRouter("Subsonic API", consts.URLPathSubsonicAPI, CreateSubsonicAPIRouter())
		a.MountRouter("Public Endpoints", consts.URLPathPublic, CreatePublicRouter())
		if conf.Server.LastFM.Enabled {
			a.MountRouter("LastFM Auth", consts.URLPathNativeAPI+"/lastfm", CreateLastFMRouter())
		}
		if conf.Server.ListenBrainz.Enabled {
			a.MountRouter("ListenBrainz Auth", consts.URLPathNativeAPI+"/listenbrainz", CreateListenBrainzRouter())
		}
		if conf.Server.Prometheus.Enabled {
			// blocking call because takes <1ms but useful if fails
			core.WriteInitialMetrics()
			a.MountRouter("Prometheus metrics", conf.Server.Prometheus.MetricsPath, promhttp.Handler())
		}
		if conf.Server.DevEnableProfiler {
			a.MountRouter("Profiling", "/debug", middleware.Profiler())
		}
		if strings.HasPrefix(conf.Server.UILoginBackgroundURL, "/") {
			a.MountRouter("Background images", consts.DefaultUILoginBackgroundURL, backgrounds.NewHandler())
		}
		return a.Run(ctx, conf.Server.Address, conf.Server.Port, conf.Server.TLSCert, conf.Server.TLSKey)
	}
}

func schedulePeriodicScan(ctx context.Context) func() error {
	return func() error {
		schedule := conf.Server.ScanSchedule
		if schedule == "" {
			log.Warn("Periodic scan is DISABLED")
			return nil
		}

		scanner := GetScanner()
		schedulerInstance := scheduler.GetInstance()

		log.Info("Scheduling periodic scan", "schedule", schedule)
		err := schedulerInstance.Add(schedule, func() {
			_ = scanner.RescanAll(ctx, false)
		})
		if err != nil {
			log.Error("Error scheduling periodic scan", err)
		}

		time.Sleep(2 * time.Second) // Wait 2 seconds before the initial scan
		log.Debug("Executing initial scan")
		if err := scanner.RescanAll(ctx, false); err != nil {
			log.Error("Error executing initial scan", err)
		}
		log.Debug("Finished initial scan")
		return nil
	}
}

func startScheduler(ctx context.Context) func() error {
	log.Info(ctx, "Starting scheduler")
	schedulerInstance := scheduler.GetInstance()

	return func() error {
		schedulerInstance.Run(ctx)
		return nil
	}
}

func startPlaybackServer(ctx context.Context) func() error {
	log.Info(ctx, "Starting playback server")

	playbackInstance := playback.GetInstance()

	return func() error {
		return playbackInstance.Run(ctx)
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
	rootCmd.PersistentFlags().String("datafolder", viper.GetString("datafolder"), "folder to store application data (DB), needs write access")
	rootCmd.PersistentFlags().String("cachefolder", viper.GetString("cachefolder"), "folder to store cache data (transcoding, images...), needs write access")
	rootCmd.PersistentFlags().StringP("loglevel", "l", viper.GetString("loglevel"), "log level, possible values: error, info, debug, trace")

	_ = viper.BindPFlag("musicfolder", rootCmd.PersistentFlags().Lookup("musicfolder"))
	_ = viper.BindPFlag("datafolder", rootCmd.PersistentFlags().Lookup("datafolder"))
	_ = viper.BindPFlag("cachefolder", rootCmd.PersistentFlags().Lookup("cachefolder"))
	_ = viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))

	rootCmd.Flags().StringP("address", "a", viper.GetString("address"), "IP address to bind to")
	rootCmd.Flags().IntP("port", "p", viper.GetInt("port"), "HTTP port Navidrome will listen to")
	rootCmd.Flags().String("baseurl", viper.GetString("baseurl"), "base URL to configure Navidrome behind a proxy (ex: /music or http://my.server.com)")
	rootCmd.Flags().String("tlscert", viper.GetString("tlscert"), "optional path to a TLS cert file (enables HTTPS listening)")
	rootCmd.Flags().String("unixsocketperm", viper.GetString("unixsocketperm"), "optional file permission for the unix socket")
	rootCmd.Flags().String("tlskey", viper.GetString("tlskey"), "optional path to a TLS key file (enables HTTPS listening)")

	rootCmd.Flags().Duration("sessiontimeout", viper.GetDuration("sessiontimeout"), "how long Navidrome will wait before closing web ui idle sessions")
	rootCmd.Flags().Duration("scaninterval", viper.GetDuration("scaninterval"), "how frequently to scan for changes in your music library")
	rootCmd.Flags().String("uiloginbackgroundurl", viper.GetString("uiloginbackgroundurl"), "URL to a backaground image used in the Login page")
	rootCmd.Flags().Bool("enabletranscodingconfig", viper.GetBool("enabletranscodingconfig"), "enables transcoding configuration in the UI")
	rootCmd.Flags().String("transcodingcachesize", viper.GetString("transcodingcachesize"), "size of transcoding cache")
	rootCmd.Flags().String("imagecachesize", viper.GetString("imagecachesize"), "size of image (art work) cache. set to 0 to disable cache")
	rootCmd.Flags().Bool("autoimportplaylists", viper.GetBool("autoimportplaylists"), "enable/disable .m3u playlist auto-import`")

	rootCmd.Flags().Bool("prometheus.enabled", viper.GetBool("prometheus.enabled"), "enable/disable prometheus metrics endpoint`")
	rootCmd.Flags().String("prometheus.metricspath", viper.GetString("prometheus.metricspath"), "http endpoint for prometheus metrics")

	_ = viper.BindPFlag("address", rootCmd.Flags().Lookup("address"))
	_ = viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("tlscert", rootCmd.Flags().Lookup("tlscert"))
	_ = viper.BindPFlag("unixsocketperm", rootCmd.Flags().Lookup("unixsocketperm"))
	_ = viper.BindPFlag("tlskey", rootCmd.Flags().Lookup("tlskey"))
	_ = viper.BindPFlag("baseurl", rootCmd.Flags().Lookup("baseurl"))

	_ = viper.BindPFlag("sessiontimeout", rootCmd.Flags().Lookup("sessiontimeout"))
	_ = viper.BindPFlag("scaninterval", rootCmd.Flags().Lookup("scaninterval"))
	_ = viper.BindPFlag("uiloginbackgroundurl", rootCmd.Flags().Lookup("uiloginbackgroundurl"))

	_ = viper.BindPFlag("prometheus.enabled", rootCmd.Flags().Lookup("prometheus.enabled"))
	_ = viper.BindPFlag("prometheus.metricspath", rootCmd.Flags().Lookup("prometheus.metricspath"))

	_ = viper.BindPFlag("enabletranscodingconfig", rootCmd.Flags().Lookup("enabletranscodingconfig"))
	_ = viper.BindPFlag("transcodingcachesize", rootCmd.Flags().Lookup("transcodingcachesize"))
	_ = viper.BindPFlag("imagecachesize", rootCmd.Flags().Lookup("imagecachesize"))
}
