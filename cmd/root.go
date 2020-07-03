package cmd

import (
	"fmt"
	"os"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/db"
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
			startServer()
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

func startServer() {
	db.EnsureLatestVersion()

	subsonic, err := CreateSubsonicAPIRouter()
	if err != nil {
		panic(fmt.Sprintf("Could not create the Subsonic API router. Aborting! err=%v", err))
	}
	a := CreateServer(conf.Server.MusicFolder)
	a.MountRouter(consts.URLPathSubsonicAPI, subsonic)
	a.MountRouter(consts.URLPathUI, CreateAppRouter())
	a.Run(fmt.Sprintf(":%d", conf.Server.Port))
}

// TODO: Implemement some struct tags to map flags to viper
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

	rootCmd.Flags().IntP("port", "p", viper.GetInt("port"), "HTTP port Navidrome will use")
	rootCmd.Flags().Duration("sessiontimeout", viper.GetDuration("sessiontimeout"), "how long Navidrome will wait before closing web ui idle sessions")
	rootCmd.Flags().Duration("scaninterval", viper.GetDuration("scaninterval"), "how frequently to scan for changes in your music library")
	rootCmd.Flags().String("baseurl", viper.GetString("baseurl"), "base URL (only the path part) to configure Navidrome behind a proxy (ex: /music)")
	rootCmd.Flags().String("uiloginbackgroundurl", viper.GetString("uiloginbackgroundurl"), "URL to a backaground image used in the Login page")
	rootCmd.Flags().Bool("enabletranscodingconfig", viper.GetBool("enabletranscodingconfig"), "enables transcoding configuration in the UI")
	rootCmd.Flags().String("transcodingcachesize", viper.GetString("transcodingcachesize"), "size of transcoding cache")
	rootCmd.Flags().String("imagecachesize", viper.GetString("imagecachesize"), "size of image (art work) cache. set to 0 to disable cache")

	_ = viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("sessiontimeout", rootCmd.Flags().Lookup("sessiontimeout"))
	_ = viper.BindPFlag("scaninterval", rootCmd.Flags().Lookup("scaninterval"))
	_ = viper.BindPFlag("baseurl", rootCmd.Flags().Lookup("baseurl"))
	_ = viper.BindPFlag("uiloginbackgroundurl", rootCmd.Flags().Lookup("uiloginbackgroundurl"))
	_ = viper.BindPFlag("enabletranscodingconfig", rootCmd.Flags().Lookup("enabletranscodingconfig"))
	_ = viper.BindPFlag("transcodingcachesize", rootCmd.Flags().Lookup("transcodingcachesize"))
	_ = viper.BindPFlag("imagecachesize", rootCmd.Flags().Lookup("imagecachesize"))
}
