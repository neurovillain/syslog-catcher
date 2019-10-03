package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	target = flag.String("target", "127.0.0.1:61614", "grpc server port")

	clients = []struct {
		Name   string
		Events []pb.EventType
		Nets   []string
	}{
		{
			Name:   "RecvPortUpDownIn192.168.1.0/24",
			Events: []pb.EventType{pb.EventType_PortUp, pb.EventType_PortDown},
			Nets:   []string{"192.168.1.0/24"},
		},
		{
			Name:   "RecvLoopDetectIn192.168.1.0/24",
			Events: []pb.EventType{pb.EventType_PortLoopDetect},
			Nets:   []string{"192.168.1.0/24"},
		},
		{
			Name:   "RecvPortUpDownIn172.16.0.0/24",
			Events: []pb.EventType{pb.EventType_PortUp, pb.EventType_PortDown},
			Nets:   []string{"172.16.0.0/24"},
		},
		{
			Name:   "RecvLoopDetectIn172.16.0.0/24",
			Events: []pb.EventType{pb.EventType_PortLoopDetect},
			Nets:   []string{"172.16.0.0/24"},
		},
	}
)

func init() {
	flag.Parse()
}

func main() {
	closeCh := make(chan struct{})

	wg := &sync.WaitGroup{}
	wg.Add(len(clients))

	for _, v := range clients {
		c, err := newClient(*target, v.Name, v.Events, v.Nets, wg, closeCh)
		if err != nil {
			log.Fatal(err)
		}
		go c.read()
	}
	go func() {
		cmd := make(chan os.Signal)
		signal.Notify(cmd, syscall.SIGINT, syscall.SIGTERM)
		log.Debug((<-cmd).String())
		close(closeCh)
	}()

	wg.Wait()
}

type client struct {
	name    string
	stream  pb.SyslogCatcher_EventsClient
	waitGr  *sync.WaitGroup
	closeCh chan struct{}
	recv    int
	done    bool
}

// newClient - create new service client with specified parameters.
func newClient(server, name string, types []pb.EventType, nets []string, wg *sync.WaitGroup, closeCh chan struct{}) (*client, error) {
	grpcConn, err := grpc.Dial(server, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	grpcClient := pb.NewSyslogCatcherClient(grpcConn)
	grpcStream, err := grpcClient.Events(context.Background(), &pb.EventRequest{
		ClientName: name,
		Events:     types,
		Nets:       nets,
	})
	if err != nil {
		return nil, err
	}

	return &client{
		name:    name,
		stream:  grpcStream,
		waitGr:  wg,
		closeCh: closeCh,
	}, nil
}

// read - start recieve message from catcher service.
func (c *client) read() {
	defer func() {
		c.stream.CloseSend()
		c.waitGr.Done()
	}()
	go c.update()
	for !c.done {
		event, err := c.stream.Recv()
		if err != nil {
			log.Infof("stream closed by server")
			c.done = true
			return
		}
		c.recv++
		fmt.Println("client", c.name, "recv", event)
	}
}

// update - control client state.
func (c *client) update() {
	for !c.done {
		after := time.After(time.Duration(10) * time.Second)
		select {
		case <-after:
			fmt.Printf("client %s recv %d messages\n", c.name, c.recv)
		case <-c.closeCh:
			c.done = true
			return
		}
	}
}
