package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// storage - хранилище счетчиков событий портов.
type storage struct {
	portMu *sync.Mutex
	ports  map[string]int
}

// newStorage - создать новый экземпляр общего хранилища данных.
func newStorage() *storage {
	return &storage{
		portMu: &sync.Mutex{},
		ports:  make(map[string]int),
	}
}

// updateCounter - обновить счетчик событий по заданному порту.
func (s *storage) updateCounter(host string, port uint32) {
	s.portMu.Lock()
	defer s.portMu.Unlock()
	s.ports[fmt.Sprintf("%s~%d", host, port)]++
}

// getCounter - вернуть количество событий по заданному порту.
func (s *storage) getCounter(host string, port uint32) int {
	s.portMu.Lock()
	defer s.portMu.Unlock()
	if n, exist := s.ports[fmt.Sprintf("%s~%d", host, port)]; exist {
		return n
	}
	return 0
}

// client - реализация клиентского подключения к сервису syslog-catcher.
type client struct {
	name    string
	store   *storage
	waitGr  *sync.WaitGroup
	closeCh chan struct{}
	stream  pb.SyslogCatcher_EventsClient
	last    *pb.Event
	recv    int
	done    bool
}

// newClient - создать новый клиент GRPC с указанными параметрами.
// клиент подписывается на рассылку сообщений указанного типа для указанного набора сетей.
func newClient(server, name string, store *storage, types []pb.EventType, nets []string, wg *sync.WaitGroup, closeCh chan struct{}) (*client, error) {
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
		store:   store,
		stream:  grpcStream,
		waitGr:  wg,
		closeCh: closeCh,
	}, nil
}

// read - подключиться к сервису и считать данные.
func (c *client) read() {
	defer func() {
		c.stream.CloseSend()
		c.waitGr.Done()
	}()
	go c.update() // Вспомогательная функция "проверки состояния"
	for !c.done {
		event, err := c.stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Infof("stream closed by server")
				c.done = true
			}
			log.Warnf("recv err - %v", err)
			continue
		}
		c.store.updateCounter(event.GetHost(), event.GetPort())
		fmt.Println("client", c.name, "recv event", event)
		c.last = event
		c.recv++
	}
}

// update - обновить состояние клиента.
// каждые 10 секунд выводит количество событий
// для последнего полученного порта устройства.
func (c *client) update() {
	for {
		after := time.After(time.Duration(10) * time.Second)
		select {
		case <-after:
			{
				if c.last != nil {
					n := c.store.getCounter(c.last.GetHost(), c.last.GetPort())
					fmt.Printf("update - get %d events for host: %s port: %d\n", n, c.last.GetHost(), c.last.GetPort())
				}
			}
		case <-c.closeCh:
			c.done = true
			return
		}
	}
}

var (
	// Параметры клиентских "сервисов"
	clients = []struct {
		Name   string
		Events []pb.EventType
		Nets   []string
	}{
		{
			Name:   "PortUpDownIn192.168.0.0/24",
			Events: []pb.EventType{pb.EventType_PortUp, pb.EventType_PortDown},
			Nets:   []string{"192.168.0.0/24"},
		},
		{
			Name:   "LoopDetectIn192.168.0.0/24",
			Events: []pb.EventType{pb.EventType_PortLoopDetect},
			Nets:   []string{"192.168.0.0/24"},
		},
		{
			Name:   "PortUpDownIn10.0.0.0/24",
			Events: []pb.EventType{pb.EventType_PortUp, pb.EventType_PortDown},
			Nets:   []string{"10.0.0.0/24"},
		},
		{
			Name:   "LoopDetectIn10.0.0.0/24",
			Events: []pb.EventType{pb.EventType_PortLoopDetect},
			Nets:   []string{"10.0.0.0/24"},
		},
	}

	// target - адрес GRPC-сервера.
	target = flag.String("target", "127.0.0.1:61614", "grpc server port")
)

func init() {
	flag.Parse()
}

func main() {

	store := newStorage() // инициализация общего хранилища данных (счетчиков)

	closeCh := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(len(clients))

	for _, v := range clients {
		c, err := newClient(*target, v.Name, store, v.Events, v.Nets, wg, closeCh)
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
