package catcher

import (
	"fmt"
	"net"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
)

// subscriber - подписчик на рассылку сообщений.
type subscriber struct {
	name   string
	stream chan *pb.Event
	events map[pb.EventType]struct{}
	nets   []*net.IPNet
}

// newSubscriber - создать новый экземпляр подписчика на сообщения.
// events -  requested event-types,
// nets - networks for processing.
func newSubscriber(name string, events []pb.EventType, nets []string) (*subscriber, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("create subscriber - no events for service %s", name)
	}
	c := &subscriber{
		name:   name,
		stream: make(chan *pb.Event, 1024),
		events: make(map[pb.EventType]struct{}),
		nets:   make([]*net.IPNet, 0),
	}
	for _, e := range events {
		c.events[e] = struct{}{}
	}
	for _, n := range nets {
		_, nwk, err := net.ParseCIDR(n)
		if err != nil {
			return nil, err
		}
		c.nets = append(c.nets, nwk)
	}
	return c, nil
}

// pull - передать сообщение подписчику.
func (c *subscriber) pull(msg *pb.Event) {
	if len(c.events) != 0 {
		if _, ok := c.events[msg.Type]; !ok {
			return
		}
	}
	if len(c.nets) != 0 {
		found := false
		addr := net.ParseIP(msg.Host)
		if addr == nil {
			return
		}
		for _, nwk := range c.nets {
			if nwk.Contains(addr) {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}

	c.stream <- msg
}
