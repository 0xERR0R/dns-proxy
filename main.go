package main

import (
	_ "embed"
	"fmt"
	"github.com/0xERR0R/dns-proxy/cert"
	"github.com/0xERR0R/dns-proxy/config"
	"github.com/0xERR0R/dns-proxy/server"
	"github.com/cristalhq/aconfig"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"os"
	"os/signal"
	"syscall"
)

var (
	//go:embed banner.txt
	banner string
)

func configureLogging(cfg config.ProxyConfig) {
	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Errorf("invalid log level, using info as fallback")
		level = log.InfoLevel
	}
	log.SetLevel(level)
	logFormatter := &prefixed.TextFormatter{
		ForceColors:      true,
		DisableColors:    true,
		ForceFormatting:  true,
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05",
		QuoteEmptyFields: true,
	}

	logFormatter.SetColorScheme(&prefixed.ColorScheme{
		PrefixStyle:    "blue+b",
		TimestampStyle: "white+h",
	})

	log.SetFormatter(logFormatter)
}

func loadConfig() (config.ProxyConfig, error) {
	var cfg config.ProxyConfig

	loader := aconfig.LoaderFor(&cfg, aconfig.Config{
		AllFieldRequired: true,
	})

	if err := loader.Load(); err != nil {
		log.Infof("USAGE: ")
		loader.WalkFields(func(f aconfig.Field) bool {
			log.Infof("Env: %q (default value:  %q) - %s", f.Tag("env"), f.Tag("default"), f.Tag("usage"))
			return true
		})
		return cfg, err
	}
	return cfg, nil
}

func main() {
	fmt.Println(banner)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal("can't create config: ", err)
	}
	configureLogging(cfg)

	log.Infof("using config: \n%+v\n", cfg)

	repo := cert.NewFileRepo(cfg.CertDir)

	manager := cert.NewLegoManager(repo, cfg)

	certHolder := cert.NewRefreshingCertHolder(manager)

	srv, err := server.NewServer(certHolder, cfg)
	if err != nil {
		log.Fatal("can't start server")
	}

	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	srv.Start()

	go func() {
		<-signals
		log.Infof("Terminating...")
		srv.Stop()
		done <- true
	}()

	<-done
}
