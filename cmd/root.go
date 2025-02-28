package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/navidrome/navidrome/adapters/taglib"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/scheduler"
	"github.com/navidrome/navidrome/server/backgrounds"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
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
			runNavidrome(cmd.Context())
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			postRun()
		},
		Version: consts.Version,
	}
)

// Execute runs the root cobra command, which will start the Navidrome server by calling the runNavidrome function.
func Execute() {
	ctx, cancel := mainContext(context.Background())
	defer cancel()

	rootCmd.SetVersionTemplate(`{{println .Version}}`)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}

func preRun() {
	if !noBanner {
		println(resources.Banner())
	}
	conf.Load(noBanner)
}

func postRun() {
	log.Info("Navidrome stopped, bye.")
}

// runNavidrome is the main entry point for the Navidrome server. It starts all the services and blocks.
// If any of the services returns an error, it will log it and exit. If the process receives a signal to exit,
// it will cancel the context and exit gracefully.
func runNavidrome(ctx context.Context) {
	defer db.Init(ctx)()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(startServer(ctx))
	g.Go(startSignaller(ctx))
	g.Go(startScheduler(ctx))
	g.Go(startPlaybackServer(ctx))
	g.Go(schedulePeriodicBackup(ctx))
	g.Go(startInsightsCollector(ctx))
	g.Go(scheduleDBOptimizer(ctx))
	if conf.Server.Scanner.Enabled {
		g.Go(runInitialScan(ctx))
		g.Go(startScanWatcher(ctx))
		g.Go(schedulePeriodicScan(ctx))
	} else {
		log.Warn(ctx, "Automatic Scanning is DISABLED")
	}

	if err := g.Wait(); err != nil {
		log.Error("Fatal error in Navidrome. Aborting", err)
	}
}

// mainContext returns a context that is cancelled when the process receives a signal to exit.
func mainContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGABRT,
	)
}

// startServer starts the Navidrome web server, adding all the necessary routers.
func startServer(ctx context.Context) func() error {
	return func() error {
		a := CreateServer()
		a.MountRouter("Native API", consts.URLPathNativeAPI, CreateNativeAPIRouter())
		a.MountRouter("Subsonic API", consts.URLPathSubsonicAPI, CreateSubsonicAPIRouter(ctx))
		a.MountRouter("Public Endpoints", consts.URLPathPublic, CreatePublicRouter())
		if conf.Server.LastFM.Enabled {
			a.MountRouter("LastFM Auth", consts.URLPathNativeAPI+"/lastfm", CreateLastFMRouter())
		}
		if conf.Server.ListenBrainz.Enabled {
			a.MountRouter("ListenBrainz Auth", consts.URLPathNativeAPI+"/listenbrainz", CreateListenBrainzRouter())
		}
		if conf.Server.Prometheus.Enabled {
			p := CreatePrometheus()
			// blocking call because takes <100ms but useful if fails
			p.WriteInitialMetrics(ctx)
			a.MountRouter("Prometheus metrics", conf.Server.Prometheus.MetricsPath, p.GetHandler())
		}
		if conf.Server.DevEnableProfiler {
			a.MountRouter("Profiling", "/debug", middleware.Profiler())
		}
		if strings.HasPrefix(conf.Server.UILoginBackgroundURL, "/") {
			a.MountRouter("Background images", conf.Server.UILoginBackgroundURL, backgrounds.NewHandler())
		}
		return a.Run(ctx, conf.Server.Address, conf.Server.Port, conf.Server.TLSCert, conf.Server.TLSKey)
	}
}

// schedulePeriodicScan schedules a periodic scan of the music library, if configured.
func schedulePeriodicScan(ctx context.Context) func() error {
	return func() error {
		schedule := conf.Server.Scanner.Schedule
		if schedule == "" {
			log.Warn(ctx, "Periodic scan is DISABLED")
			return nil
		}

		s := CreateScanner(ctx)
		schedulerInstance := scheduler.GetInstance()

		log.Info("Scheduling periodic scan", "schedule", schedule)
		err := schedulerInstance.Add(schedule, func() {
			_, err := s.ScanAll(ctx, false)
			if err != nil {
				log.Error(ctx, "Error executing periodic scan", err)
			}
		})
		if err != nil {
			log.Error(ctx, "Error scheduling periodic scan", err)
		}
		return nil
	}
}

func pidHashChanged(ds model.DataStore) (bool, error) {
	pidAlbum, err := ds.Property(context.Background()).DefaultGet(consts.PIDAlbumKey, "")
	if err != nil {
		return false, err
	}
	pidTrack, err := ds.Property(context.Background()).DefaultGet(consts.PIDTrackKey, "")
	if err != nil {
		return false, err
	}
	return !strings.EqualFold(pidAlbum, conf.Server.PID.Album) || !strings.EqualFold(pidTrack, conf.Server.PID.Track), nil
}

func runInitialScan(ctx context.Context) func() error {
	return func() error {
		ds := CreateDataStore()
		fullScanRequired, err := ds.Property(ctx).DefaultGet(consts.FullScanAfterMigrationFlagKey, "0")
		if err != nil {
			return err
		}
		inProgress, err := ds.Library(ctx).ScanInProgress()
		if err != nil {
			return err
		}
		pidHasChanged, err := pidHashChanged(ds)
		if err != nil {
			return err
		}
		scanNeeded := conf.Server.Scanner.ScanOnStartup || inProgress || fullScanRequired == "1" || pidHasChanged
		time.Sleep(2 * time.Second) // Wait 2 seconds before the initial scan
		if scanNeeded {
			scanner := CreateScanner(ctx)
			switch {
			case fullScanRequired == "1":
				log.Warn(ctx, "Full scan required after migration")
				_ = ds.Property(ctx).Delete(consts.FullScanAfterMigrationFlagKey)
			case pidHasChanged:
				log.Warn(ctx, "PID config changed, performing full scan")
				fullScanRequired = "1"
			case inProgress:
				log.Warn(ctx, "Resuming interrupted scan")
			default:
				log.Info("Executing initial scan")
			}

			_, err = scanner.ScanAll(ctx, fullScanRequired == "1")
			if err != nil {
				log.Error(ctx, "Scan failed", err)
			} else {
				log.Info(ctx, "Scan completed")
			}
		} else {
			log.Debug(ctx, "Initial scan not needed")
		}
		return nil
	}
}

func startScanWatcher(ctx context.Context) func() error {
	return func() error {
		if conf.Server.Scanner.WatcherWait == 0 {
			log.Debug("Folder watcher is DISABLED")
			return nil
		}
		w := CreateScanWatcher(ctx)
		err := w.Run(ctx)
		if err != nil {
			log.Error("Error starting watcher", err)
		}
		return nil
	}
}

func schedulePeriodicBackup(ctx context.Context) func() error {
	return func() error {
		schedule := conf.Server.Backup.Schedule
		if schedule == "" {
			log.Warn(ctx, "Periodic backup is DISABLED")
			return nil
		}

		schedulerInstance := scheduler.GetInstance()

		log.Info("Scheduling periodic backup", "schedule", schedule)
		err := schedulerInstance.Add(schedule, func() {
			start := time.Now()
			path, err := db.Backup(ctx)
			elapsed := time.Since(start)
			if err != nil {
				log.Error(ctx, "Error backing up database", "elapsed", elapsed, err)
				return
			}
			log.Info(ctx, "Backup complete", "elapsed", elapsed, "path", path)

			count, err := db.Prune(ctx)
			if err != nil {
				log.Error(ctx, "Error pruning database", "error", err)
			} else if count > 0 {
				log.Info(ctx, "Successfully pruned old files", "count", count)
			} else {
				log.Info(ctx, "No backups pruned")
			}
		})

		return err
	}
}

func scheduleDBOptimizer(ctx context.Context) func() error {
	return func() error {
		log.Info(ctx, "Scheduling DB optimizer", "schedule", consts.OptimizeDBSchedule)
		schedulerInstance := scheduler.GetInstance()
		err := schedulerInstance.Add(consts.OptimizeDBSchedule, func() {
			if scanner.IsScanning() {
				log.Debug(ctx, "Skipping DB optimization because a scan is in progress")
				return
			}
			db.Optimize(ctx)
		})
		return err
	}
}

// startScheduler starts the Navidrome scheduler, which is used to run periodic tasks.
func startScheduler(ctx context.Context) func() error {
	return func() error {
		log.Info(ctx, "Starting scheduler")
		schedulerInstance := scheduler.GetInstance()
		schedulerInstance.Run(ctx)
		return nil
	}
}

// startInsightsCollector starts the Navidrome Insight Collector, if configured.
func startInsightsCollector(ctx context.Context) func() error {
	return func() error {
		if !conf.Server.EnableInsightsCollector {
			log.Info(ctx, "Insight Collector is DISABLED")
			return nil
		}
		log.Info(ctx, "Starting Insight Collector")
		select {
		case <-time.After(conf.Server.DevInsightsInitialDelay):
		case <-ctx.Done():
			return nil
		}
		ic := CreateInsights()
		ic.Run(ctx)
		return nil
	}
}

// startPlaybackServer starts the Navidrome playback server, if configured.
// It is responsible for the Jukebox functionality
func startPlaybackServer(ctx context.Context) func() error {
	return func() error {
		if !conf.Server.Jukebox.Enabled {
			log.Debug("Jukebox is DISABLED")
			return nil
		}
		log.Info(ctx, "Starting Jukebox service")
		playbackInstance := GetPlaybackServer()
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
	rootCmd.PersistentFlags().String("logfile", viper.GetString("logfile"), "log file path, if not set logs will be printed to stderr")

	_ = viper.BindPFlag("musicfolder", rootCmd.PersistentFlags().Lookup("musicfolder"))
	_ = viper.BindPFlag("datafolder", rootCmd.PersistentFlags().Lookup("datafolder"))
	_ = viper.BindPFlag("cachefolder", rootCmd.PersistentFlags().Lookup("cachefolder"))
	_ = viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))
	_ = viper.BindPFlag("logfile", rootCmd.PersistentFlags().Lookup("logfile"))

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
	rootCmd.Flags().String("albumplaycountmode", viper.GetString("albumplaycountmode"), "how to compute playcount for albums. absolute (default) or normalized")
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
