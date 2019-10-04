package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/neurovillain/syslog-catcher/pkg/service/catcher"
	"github.com/neurovillain/syslog-catcher/pkg/service/config"
	log "github.com/sirupsen/logrus"
)

var (
	// cfile - параметр запуска приложения - путь к файлу конфигурации.
	cfile = flag.String("config", "service_config.yml", "service configuration file path")
)

func init() {
	flag.Parse()
}

func main() {
	cfg, err := config.ParseFile(*cfile)
	if err != nil {
		log.Fatal(err)
	}

	service, err := catcher.NewService(cfg)
	if err != nil {
		log.Fatal(err)
	}

	go service.Serve()

	cmd := make(chan os.Signal)
	signal.Notify(cmd, syscall.SIGINT, syscall.SIGTERM)
	log.Debug((<-cmd).String())

	service.Close()
}
