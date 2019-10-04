package syslog

import (
	"fmt"
	"net"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
	"github.com/neurovillain/syslog-catcher/pkg/service/parser"
	log "github.com/sirupsen/logrus"
)

// Listener - интерфейс приема входящих сообщений SYSLOG.
type Listener interface {
	// Listen - запустить основной цикл - занять UDP порт,
	// в цикле обработать сообщения и передать их в канал retCh.
	Listen(chan *pb.Event)

	// Counters - вернуть текущее состояние счетчиков.
	Counters() (int, int)

	// Close - завершить работу и закрыть соеднинение.
	Close()
}

// NewListener - создать новый экземпляр Listener.
func NewListener(addr string, bufSize int, parser parser.Parser) (Listener, error) {
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

// listener - реализация интерфейса Listener.
type listener struct {
	bufSize int
	parser  parser.Parser
	conn    *net.UDPConn
	done    bool
	result  chan *pb.Event

	// Счетчики для отладки
	recv   int
	parsed int
}

// Listen - запустить основной цикл - занять UDP порт,
// в цикле обработать сообщения и передать их в канал retCh.
func (l *listener) Listen(ch chan *pb.Event) {
	l.result = ch
	buf := make([]byte, l.bufSize)
	for {
		n, _, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			netOpError, ok := err.(*net.OpError)
			if ok && netOpError.Err.Error() == "use of closed network connection" {
				return
			}
			log.Debugf("listener recv err - %v", err)
			continue
		}
		go l.handle(string(buf[:n]))
	}
}

// handle - обработать полученное сообщение и направить его в канал.
func (l *listener) handle(message string) {
	l.recv++
	if event, err := l.parser.Parse(message); err == nil {
		l.parsed++
		l.result <- event
	}
}

// Counters - вернуть текущее состояние счетчиков.
func (l *listener) Counters() (int, int) {
	return l.recv, l.parsed
}

// Close - завершить работу и закрыть соеднинение.
func (l *listener) Close() {
	l.conn.Close()
	log.Debugf("total: recv - %d, parsed - %d", l.recv, l.parsed)
}
