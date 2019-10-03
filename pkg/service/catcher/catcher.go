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
	"github.com/neurovillain/syslog-catcher/pkg/service/syslog"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Service - interface of syslog-catcher service.
type Service interface {
	// Serve - run main exec loop, start listen syslog and recv client message.
	Serve()

	// Close - stop handle and close all connections.
	Close()
}

// NewService - create new instance of syslog catcher service.
func NewService(cfg *config.Config) (Service, error) {
	loglevel, err := log.ParseLevel(cfg.Log.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level err - %v", err)
	}
	log.SetLevel(loglevel)

	f, err := os.OpenFile(cfg.Log.File, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("init log file \"%s\" err - %v", cfg.Log.File, err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))

	parser, err := syslog.NewParser(cfg.Syslog.Templates)
	if err != nil {
		return nil, err
	}
	lsn, err := syslog.NewListener(cfg.Syslog.Listen, cfg.Syslog.BufSize, parser)
	if err != nil {
		return nil, err
	}

	conn, err := net.Listen("tcp", cfg.GRPC.Listen)
	if err != nil {
		return nil, err
	}
	server := grpc.NewServer()
	if err != nil {
		return nil, err
	}
	log.Infof("listen grpc requests on %s", cfg.GRPC.Listen)

	s := &service{
		server:      server,
		conn:        conn,
		listener:    lsn,
		subsMu:      sync.Mutex{},
		subscribers: make(map[string]*subscriber),
		closed:      make(chan struct{}),
	}
	pb.RegisterSyslogCatcherServer(server, s)

	return s, nil
}

// service - implementation of syslog catcher service.
type service struct {
	server      *grpc.Server
	conn        net.Listener
	listener    syslog.Listener
	subsMu      sync.Mutex
	subscribers map[string]*subscriber
	closed      chan struct{}
}

// Serve - run main exec loop.
func (s *service) Serve() {

	go s.server.Serve(s.conn)
	defer s.server.GracefulStop()

	ch := make(chan *pb.Event)
	go s.listener.Listen(ch)
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

// Events - (implements GRPC method) - send data to client.
func (s *service) Events(rq *pb.EventRequest, stream pb.SyslogCatcher_EventsServer) error {
	sub, err := newSubscriber(rq.GetClientName(), rq.GetEvents(), rq.GetNets())
	if err != nil {
		return err
	}
	uid := fmt.Sprintf("%s-%d", sub.name, time.Now().Unix())

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
		after := time.After(time.Duration(3) * time.Minute)
		select {
		case <-after:
			return nil
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

// Close - stop syslog catcher service.
func (s *service) Close() {
	s.listener.Close()
	s.conn.Close()
	close(s.closed)
	log.Info("----- syslog catcher service is stopped -----")
}
