package test

import (
	"fmt"
	"testing"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
	"github.com/neurovillain/syslog-catcher/pkg/service/syslog"
)

var (
	patterns = []string{
		"link_up ~ $device_addr$ - - - port $device_port$ change link state to up with $port_speed$ $port_duplex$",
		"link_down ~ $device_addr$ - - - port $device_port$ change link state to down",
		"loopdetect ~ $device_addr$ - - - port $device_port$ disabled by loop detect service",
	}

	messages = []struct {
		Text   string
		Type   pb.EventType
		Host   string
		Port   uint32
		Speed  pb.PortSpeed
		Duplex pb.PortDuplex
		OK     bool
	}{
		{
			Text:   "192.168.1.99 - - - port 1 change link state to up with 100mb half-duplex",
			Type:   pb.EventType_PortUp,
			Host:   "192.168.1.99",
			Port:   1,
			Speed:  pb.PortSpeed_Speed100Mb,
			Duplex: pb.PortDuplex_Half,
			OK:     true,
		},
		{
			Text:   "192.168.1.100 - - - port 2 change link state to up with 10mb half-duplex",
			Type:   pb.EventType_PortUp,
			Host:   "192.168.1.100",
			Port:   2,
			Speed:  pb.PortSpeed_Speed10Mb,
			Duplex: pb.PortDuplex_Half,
			OK:     true,
		},
		{
			Text:   "192.168.1.101 - - - port ethernet1/0/3 change link state to up with 1000mb full-duplex",
			Type:   pb.EventType_PortUp,
			Host:   "192.168.1.101",
			Port:   3,
			Speed:  pb.PortSpeed_Speed1Gb,
			Duplex: pb.PortDuplex_Full,
			OK:     true,
		},
		{
			Text: "192.168.1. - - - port 10 change link state to up with 10mb half-duplex",
			OK:   false,
		},
		{
			Text: "192.168.1.103 - - - port X change link state to up with 10mb half-duplex",
			OK:   false,
		},
		{
			Text: "192.168.1.104 - - - port 6 change link state to up with 999mb half-duplex",
			OK:   false,
		},
		{
			Text: "192.168.1.105 - - - port 7 change link state to down",
			Type: pb.EventType_PortDown,
			Host: "192.168.1.105",
			Port: 7,
			OK:   true,
		},
		{
			Text: "192.168.1.106 - - - port eth1/8 change link state to down",
			Type: pb.EventType_PortDown,
			Host: "192.168.1.106",
			Port: 8,
			OK:   true,
		},
		{
			Text: "192.168.1.107 - - - port 9 change link state to up",
			OK:   false,
		},
		{
			Text: "192.168.1.108 - - - port 10 disabled by loop detected service",
			Type: pb.EventType_PortLoopDetect,
			Host: "192.168.1.108",
			Port: 10,
			OK:   true,
		},
		{
			Text: "192.168.1.109 - - - port X disabled by loop detected service",
			OK:   false,
		},
	}
)

func TestTextParser(t *testing.T) {

	parser, err := syslog.NewParser(patterns)
	if err != nil {
		t.Fatal(err)
	}

	for _, msg := range messages {
		event, err := parser.Parse(msg.Text)
		if err != nil {
			if msg.OK {
				t.Fatal("unexpected result - failed to parse normal message", err)
			}
			continue
		}
		if (msg.Type != event.Type) || (msg.Host != event.Host) || (msg.Port != event.Port) || (msg.Speed != event.Speed) || (msg.Duplex != event.Duplex) {
			t.Fatal("unexpected result - parse result not match with criterias", msg, event.Type, event.Host, event.Port, event.Speed, event.Duplex)
		}
		fmt.Println(event)
	}
}
