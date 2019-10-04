package test

import (
	"testing"

	"github.com/neurovillain/syslog-catcher/pkg/service/catcher"
	"github.com/neurovillain/syslog-catcher/pkg/service/config"
)

func TestSyslogCatcher(t *testing.T) {
	cfg, err := config.ParseFile("../examples/service_config.yml")
	if err != nil {
		t.Fatal(err)
	}
	s, err := catcher.NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
}
