package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	log "github.com/Sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var version = "v0.2.2"
var dirty = ""

var cfgFile string

var displayVersion string
var showVersion bool
var verbose bool
var debug bool
var check bool

func main() {
	displayVersion = fmt.Sprintf("sentry %s%s",
		version,
		dirty)
	Execute(displayVersion)
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "sentry",
	Short: "An URL monitoring alerting service",
	Long:  `Watches http/s URLs for unexpected responses.`,
	Run:   run,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
	displayVersion = version
	RootCmd.SetHelpTemplate(fmt.Sprintf("%s\nVersion:\n  github.com/gesquive/%s\n",
		RootCmd.HelpTemplate(), displayVersion))
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"Path to a specific config file (default \"./config.yml\")")
	RootCmd.PersistentFlags().StringP("log-file", "l", "",
		"Path to log file (default \"/var/log/sentry.log\")")
	RootCmd.PersistentFlags().BoolVar(&check, "check", false,
		"Check the config for errors and exit")
	RootCmd.PersistentFlags().BoolP("run-once", "o", false,
		"Run once and exit")
	RootCmd.PersistentFlags().BoolP("no-alerts", "n", false,
		"Disable all outgoing email alerts, log alerts only")
	RootCmd.PersistentFlags().BoolVar(&showVersion, "version", false,
		"Display the version number and exit")

	RootCmd.PersistentFlags().StringP("smtp-server", "x", "localhost",
		"The SMTP server to send email through")
	RootCmd.PersistentFlags().Uint32P("smtp-port", "r", 25,
		"The port to use for the SMTP server")
	RootCmd.PersistentFlags().StringP("smtp-username", "u", "",
		"Authenticate the SMTP server with this user")
	RootCmd.PersistentFlags().StringP("smtp-password", "w", "",
		"Authenticate the SMTP server with this password")

	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Print logs to stdout instead of file")

	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "D", false,
		"Include debug statements in log output")
	RootCmd.PersistentFlags().MarkHidden("debug")

	viper.SetEnvPrefix("sentry")
	viper.AutomaticEnv()
	viper.BindEnv("log_file")
	viper.BindEnv("no_alerts")
	viper.BindEnv("run_once")
	viper.BindEnv("smtp_server")
	viper.BindEnv("smtp_port")
	viper.BindEnv("smtp_username")
	viper.BindEnv("smtp_password")

	viper.BindPFlag("log_file", RootCmd.PersistentFlags().Lookup("log-file"))
	viper.BindPFlag("no_alerts", RootCmd.PersistentFlags().Lookup("no-alerts"))
	viper.BindPFlag("run_once", RootCmd.PersistentFlags().Lookup("run-once"))
	viper.BindPFlag("smtp.server", RootCmd.PersistentFlags().Lookup("smtp-server"))
	viper.BindPFlag("smtp.port", RootCmd.PersistentFlags().Lookup("smtp-port"))
	viper.BindPFlag("smtp.username", RootCmd.PersistentFlags().Lookup("smtp-username"))
	viper.BindPFlag("smtp.password", RootCmd.PersistentFlags().Lookup("smtp-password"))

	viper.SetDefault("log_file", "/var/log/sentry.log")
	viper.SetDefault("no_alerts", false)
	viper.SetDefault("run_once", false)
	viper.SetDefault("smtp.server", "localhost")
	viper.SetDefault("smtp.port", 25)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName("config")               // name of config file (without extension)
	viper.AddConfigPath(".")                    // add current directory as first search path
	viper.AddConfigPath("$HOME/.config/sentry") // add home directory to search path
	viper.AddConfigPath("/etc/sentry")          // add etc to search path
	viper.AutomaticEnv()                        // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if !showVersion {
			log.Error("Error opening config: ", err)
		}
	}
}

func run(cmd *cobra.Command, args []string) {
	if showVersion {
		fmt.Println(displayVersion)
		os.Exit(0)
	}

	log.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat: time.RFC3339,
	})

	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	logFilePath := getLogFilePath(viper.GetString("log_file"))
	log.Debugf("config: log_file=%s", logFilePath)
	if verbose {
		log.SetOutput(os.Stdout)
	} else {
		logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening log file=%v", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	log.Infof("config: file=%s", viper.ConfigFileUsed())
	if viper.ConfigFileUsed() == "" {
		log.Fatal("No config file found.")
	}

	smtpSettings := SMTPSettings{
		viper.GetString("smtp.server"),
		viper.GetInt("smtp.port"),
		viper.GetString("smtp.username"),
		viper.GetString("smtp.password"),
	}
	log.Debugf("config: smtp={Host:%s Port:%d UserName:%s}", smtpSettings.Host,
		smtpSettings.Port, smtpSettings.UserName)

	rawTargets := viper.Get("targets").([]interface{})
	targetConfigs, err := getTargetConfigs(rawTargets, viper.Get("defaults"))
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
	for _, target := range targetConfigs {
		log.Debugf("config: target=%+v", target)
	}

	sentry := NewSentry(targetConfigs, smtpSettings, version)
	if viper.GetBool("no_alerts") {
		log.Debugf("config: no-alerts=true")
		sentry.DisableAlerts()
	}

	if check {
		log.Infof("Config file format checks out, exiting")
		if !debug {
			log.Infof("Use the --debug flag for more info")
		}
		os.Exit(0)
	}

	// finally, run the sentry monitor
	if viper.GetBool("run_once") {
		sentry.RunCheck()
	} else {
		sentry.Run()
	}
}

func getTargetConfigs(config []interface{}, defaults interface{}) ([]SentryTarget, error) {
	defaultTarget, err := NewTarget(defaults)
	if err != nil {
		log.Errorf("default values invalid - %v", err)
		return nil, err
	}
	var targets []SentryTarget
	for i, targetConfig := range config {
		target, err := defaultTarget.SpawnTarget(targetConfig)
		if err != nil {
			log.Error(err)
			log.Errorf("invalid values for target=%d", i)
		}
		targets = append(targets, *target)
	}
	return targets, nil
}

func getLogFilePath(defaultPath string) (logPath string) {
	fi, err := os.Stat(defaultPath)
	if err == nil && fi.IsDir() {
		logPath = path.Join(defaultPath, "sentry.log")
	} else {
		logPath = defaultPath
	}
	return
}
