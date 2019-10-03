package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	target = flag.String("target", "127.0.0.1:51514", "send syslog to address")
)

func init() {
	flag.Parse()
}

func main() {
	addr, err := net.ResolveUDPAddr("udp", *target)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal(err)
	}

	done := false
	go func() {
		for !done {
			_, err := conn.Write(flood())
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(time.Duration(rand.Intn(700)+300) * time.Millisecond)
		}
	}()

	cmd := make(chan os.Signal)
	signal.Notify(cmd, syscall.SIGINT, syscall.SIGTERM)
	log.Debug((<-cmd).String())

	done = true
	conn.Close()

}

// flood - create random syslog message.
func flood() []byte {
	var s string
	switch rand.Intn(10) {
	case 0:
		s = fmt.Sprintf("192.168.1.%d - - - port %d change link state to up with 10MB FULL", rand.Intn(254)+1, rand.Intn(10)+1)
	case 1:
		s = fmt.Sprintf("192.168.1.%d - - - port Eth1/0/%d change link state to up with 100MB half-duplex", rand.Intn(254)+1, rand.Intn(10)+1)
	case 2:
		s = fmt.Sprintf("192.168.1.%d - - - port %d change link state to down", rand.Intn(254)+1, rand.Intn(10)+1)
	case 3:
		s = fmt.Sprintf("192.168.1.%d - - - port %d disabled by loop detect service", rand.Intn(254)+1, rand.Intn(10)+1)
	case 4:
		s = fmt.Sprintf("172.16.0.%d - - - port Ethernet1/%d change link state to up with 100MB half-duplex", rand.Intn(254)+1, rand.Intn(10)+1)
	case 5:
		s = fmt.Sprintf("172.16.0.%d - - - port %d change link state to up with 10 full-duplex", rand.Intn(254)+1, rand.Intn(10)+1)
	case 6:
		s = fmt.Sprintf("172.16.0.%d - - - port Eth1/0/%d change link state to 100MB full-duplex", rand.Intn(254)+1, rand.Intn(10)+1)
	case 7:
		s = fmt.Sprintf("172.16.0.%d - - - port %d disabled by loop detect service", rand.Intn(254)+1, rand.Intn(10)+1)
	default:
		s = "127.0.0.1 - - - random flood message"
	}
	log.Infof("send message - %s", s)
	return []byte(s)
}
