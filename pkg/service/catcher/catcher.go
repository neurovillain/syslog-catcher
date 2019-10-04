package catcher

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
	"github.com/neurovillain/syslog-catcher/pkg/service/config"
	"github.com/neurovillain/syslog-catcher/pkg/service/parser"
	"github.com/neurovillain/syslog-catcher/pkg/service/syslog"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Service - сервис обработки входящих syslog-сообщений.
type Service interface {
	// Serve - запустить основной цикл работы сервиса -
	// получение и преобразование входящих данных и передача их подписчикам.
	Serve()

	// Close - завершить работу и закрыть все соединения.
	Close()
}

// NewService - create new instance of syslog catcher service.
func NewService(cfg *config.Config) (Service, error) {
	logLevel, err := log.ParseLevel(cfg.Log.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level err - %v", err)
	}
	log.SetLevel(logLevel)

	f, err := os.OpenFile(cfg.Log.File, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("init log file \"%s\" err - %v", cfg.Log.File, err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))

	parser, err := parser.NewParser(cfg.Syslog.Templates)
	if err != nil {
		return nil, fmt.Errorf("init parser err - %v", err)
	}
	lsn, err := syslog.NewListener(cfg.Syslog.Listen, cfg.Syslog.BufSize, parser)
	if err != nil {
		return nil, fmt.Errorf("init syslog listener err - %v", err)
	}

	conn, err := net.Listen("tcp", cfg.GRPC.Listen)
	if err != nil {
		return nil, fmt.Errorf("init grpc conn err - %v", err)
	}
	log.Debugf("listen grpc requests on %s", cfg.GRPC.Listen)

	s := &service{
		server:      grpc.NewServer(),
		conn:        conn,
		listener:    lsn,
		subsMu:      sync.Mutex{},
		subscribers: make(map[string]*subscriber),
		closed:      make(chan struct{}),
	}

	pb.RegisterSyslogCatcherServer(s.server, s)

	return s, nil
}

// service - реализация интерфейса Service.
type service struct {
	server      *grpc.Server
	conn        net.Listener
	listener    syslog.Listener
	subsMu      sync.Mutex
	subscribers map[string]*subscriber
	closed      chan struct{}
}

// Serve - запустить основной цикл работы сервиса -
// получение и преобразование входящих данных и
// передача их подписчикам.
func (s *service) Serve() {
	ch := make(chan *pb.Event)
	go s.listener.Listen(ch)
	go s.server.Serve(s.conn)
	defer s.server.GracefulStop()
	log.Info("----- syslog catcher service is launched -----")
	for {
		select {
		case <-s.closed:
			return
		case msg := <-ch:
			{
				s.subsMu.Lock()
				for _, c := range s.subscribers {
					c.pull(msg)
				}
				s.subsMu.Unlock()
			}
		}
	}
}

// Events - (реализация метода SyslogCatcherServer) - подключение нового подписчика к сервису.
func (s *service) Events(rq *pb.EventRequest, stream pb.SyslogCatcher_EventsServer) error {
	sub, err := newSubscriber(rq.GetClientName(), rq.GetEvents(), rq.GetNets())
	if err != nil {
		return err
	}
	// Добавялем время к имени подписчика, чтобы избежать совпадения имен сервисов.
	uid := fmt.Sprintf("%s~%d", sub.name, time.Now().Nanosecond())

	s.subsMu.Lock()
	s.subscribers[uid] = sub
	s.subsMu.Unlock()
	log.Infof("client %s is connected to service", sub.name)

	defer func() {
		s.subsMu.Lock()
		delete(s.subscribers, uid)
		s.subsMu.Unlock()
		log.Infof("client %s is disconect", sub.name)
	}()

	for {
		select {
		case <-s.closed:
			return nil
		case msg := <-sub.stream:
			{
				if err := stream.Send(msg); err != nil {
					return err
				}
			}
		}
	}
}

// Close - завершить работу и закрыть все соединения.
func (s *service) Close() {
	s.listener.Close()
	s.conn.Close()
	close(s.closed)
	log.Info("----- syslog catcher service is stopped -----")
}
