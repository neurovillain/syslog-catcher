package syslog

import (
	"fmt"
	"net"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
	log "github.com/sirupsen/logrus"
)

// Listener - inteface of syslog recivier
type Listener interface {
	// Listen - recv UDP data and send it do retCh.
	Listen(chan *pb.Event)

	// Close - stop processing data and quit.
	Close()
}

// NewListener - create new instance of listener interface.
func NewListener(addr string, bufSize int, parser Parser) (Listener, error) {
	if parser == nil {
		return nil, fmt.Errorf("ptr to parser is nil")
	}
	if bufSize < 1 {
		return nil, fmt.Errorf("listener buf size are invalid")
	}
	udpaddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		return nil, err
	}
	log.Infof("listen syslog messages on address %s", addr)

	return &listener{
		bufSize: bufSize,
		parser:  parser,
		conn:    conn,
	}, nil
}

// listener -- implementation of Listener interface.
type listener struct {
	bufSize int
	parser  Parser
	conn    *net.UDPConn
	done    bool

	// debug counters
	recv   int
	parsed int
}

// Listen - recv data and send it to parser ch.
func (l *listener) Listen(ch chan *pb.Event) {
	buf := make([]byte, l.bufSize)
	for !l.done {
		i, _, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			log.Warn(err)
		}
		go l.handle(ch, string(buf[:i]))
	}
	l.conn.Close()
}

// handle - process recivied message and send into channel.
func (l *listener) handle(ch chan *pb.Event, message string) {
	l.recv++
	if event, err := l.parser.Parse(message); err == nil {
		l.parsed++
		ch <- event
	}
}

// Close - halt message processing.
func (l *listener) Close() {
	l.done = true
	log.Infof("total: recv - %d, parsed - %d", l.recv, l.parsed)
}
