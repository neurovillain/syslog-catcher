package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	// target - адрес сервера-получателя текстовых сообщений.
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
		log.Fatalf("connect to syslog-catcher server failed - %v", err)
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

// flood - сгенерировать случайное сообщение по шаблону существующего или содержащее случайное количество полей "flood".
func flood() []byte {
	var s string
	switch rand.Intn(10) {
	case 0:
		s = fmt.Sprintf("%s - - - port %s change link state to up with %s", randomAddr(), randomPort(), randomSpeed())
	case 1:
		s = fmt.Sprintf("%s - - - port %s change link state to down", randomAddr(), randomPort())
	case 2:
		s = fmt.Sprintf("%s - - -  port %s disabled by loop detect service", randomAddr(), randomPort())
	case 3:
		s = fmt.Sprintf("%s info: interface %s UP %s", randomAddr(), randomPort(), randomSpeed())
	case 4:
		s = fmt.Sprintf("%s info: interface %s  DOWN", randomAddr(), randomPort())
	case 5:
		s = fmt.Sprintf("%s warn: loop detected on inteface %s", randomAddr(), randomPort())
	default:
		s = fmt.Sprintf("%s - %s", randomAddr(), strings.Repeat("flood ", rand.Intn(15)+1))
	}
	log.Infof("sending message - %s", s)
	return []byte(s)
}

func randomAddr() string {
	if rand.Intn(4)%2 == 0 {
		return fmt.Sprintf("192.168.0.%d", rand.Intn(10)+1)
	}

	return fmt.Sprintf("10.0.0.%d", rand.Intn(10)+1)
}

func randomPort() string {
	if rand.Intn(4)%2 == 0 {
		return fmt.Sprintf("Ethernet1/0/%d", rand.Intn(10)+1)
	}

	return fmt.Sprintf("%d", rand.Intn(10))
}

func randomSpeed() string {
	var speed, duplex string
	switch rand.Intn(3) {
	case 0:
		speed = "10"
	case 1:
		speed = "100"
	default:
		speed = "1000"
	}
	switch rand.Intn(2) {
	case 0:
		duplex = "full"
	case 1:
		duplex = "half"
	}
	switch rand.Intn(3) {
	case 0:
		return fmt.Sprintf("%sMB %s-duplex", speed, strings.ToUpper(duplex))
	case 1:
		return fmt.Sprintf("(speed:%s, duplex:%s)", speed, duplex)
	}

	return fmt.Sprintf("%s %s", speed, duplex)
}
