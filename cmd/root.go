package cmd

import (
	"fmt"
	"os"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "navidrome",
		Short: "Navidrome is a self-hosted music server and streamer",
		Long: `Navidrome is a self-hosted music server and streamer.
Complete documentation is available at https://www.navidrome.org/docs`,
		Run: func(cmd *cobra.Command, args []string) {
			start()
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

// TODO: Implemement some struct tags to map flags to viper
func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "configfile", "c", "", `config file (default "./navidrome.toml")`)
	rootCmd.PersistentFlags().String("musicfolder", viper.GetString("musicfolder"), "folder where your music is stored")
	rootCmd.PersistentFlags().String("datafolder", viper.GetString("datafolder"), "folder to store application data (DB, cache...), needs write access")
	rootCmd.PersistentFlags().StringP("loglevel", "l", viper.GetString("loglevel"), "log level, possible values: error, info, debug, trace")

	viper.BindPFlag("musicfolder", rootCmd.PersistentFlags().Lookup("musicfolder"))
	viper.BindPFlag("datafolder", rootCmd.PersistentFlags().Lookup("datafolder"))
	viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))

	rootCmd.Flags().StringP("port", "p", viper.GetString("port"), "HTTP port Navidrome will use")
	rootCmd.Flags().String("sessiontimeout", viper.GetString("sessiontimeout"), "how long Navidrome will wait before closing web ui idle sessions")
	rootCmd.Flags().String("scaninterval", viper.GetString("scaninterval"), "how frequently to scan for changes in your music library")
	rootCmd.Flags().String("baseurl", viper.GetString("baseurl"), "base URL (only the path part) to configure Navidrome behind a proxy (ex: /music)")
	rootCmd.Flags().String("uiloginbackgroundurl", viper.GetString("uiloginbackgroundurl"), "URL to a backaground image used in the Login page")
	rootCmd.Flags().Bool("enabletranscodingconfig", viper.GetBool("enabletranscodingconfig"), "enables transcoding configuration in the UI")
	rootCmd.Flags().String("transcodingcachesize", viper.GetString("transcodingcachesize"), "size of transcoding cache")
	rootCmd.Flags().String("imagecachesize", viper.GetString("imagecachesize"), "size of image (art work) cache. set to 0 to disable cache")

	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("sessiontimeout", rootCmd.Flags().Lookup("sessiontimeout"))
	viper.BindPFlag("scaninterval", rootCmd.Flags().Lookup("scaninterval"))
	viper.BindPFlag("baseurl", rootCmd.Flags().Lookup("baseurl"))
	viper.BindPFlag("uiloginbackgroundurl", rootCmd.Flags().Lookup("uiloginbackgroundurl"))
	viper.BindPFlag("enabletranscodingconfig", rootCmd.Flags().Lookup("enabletranscodingconfig"))
	viper.BindPFlag("transcodingcachesize", rootCmd.Flags().Lookup("transcodingcachesize"))
	viper.BindPFlag("imagecachesize", rootCmd.Flags().Lookup("imagecachesize"))
}

func initConfig() {
	conf.SetDefaults()

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in local directory with name "navidrome" (without extension).
		viper.AddConfigPath(".")
		viper.SetConfigName("navidrome")
	}

	viper.BindEnv("port")
	viper.SetEnvPrefix("ND")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error loading config file '%s'. Error: %s\n", viper.ConfigFileUsed(), err)
		os.Exit(1)
	}
}
