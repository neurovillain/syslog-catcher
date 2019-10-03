package syslog

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
	log "github.com/sirupsen/logrus"
)

// Parser - interface of syslog message parser.
type Parser interface {
	// Parse - unmarshal recv syslog message to grpc event.
	Parse(string) (*pb.Event, error)
}

// NewParser - create new parser
func NewParser(patterns []string) (Parser, error) {
	if len(patterns) == 0 {
		return nil, errors.New("no patterns for parser are provided")
	}
	result := &textParser{
		patterns: make(map[int][]*textPattern),
	}
	for _, v := range patterns {
		args := strings.SplitN(v, " ~ ", 2)
		if len(args) != 2 {
			return nil, errors.New("unknown pattern format")
		}
		pattern, err := newTextPattern(args[0], args[1])
		if err != nil {
			return nil, err
		}
		arr, exist := result.patterns[len(pattern.fields)]
		if !exist {
			arr = make([]*textPattern, 0)
		}
		arr = append(arr, pattern)
		result.patterns[len(pattern.fields)] = arr
	}
	log.Infof("defined %d text parser patterns", len(patterns))

	return result, nil
}

// textParser - implementation of Parser interface.
type textParser struct {
	patterns map[int][]*textPattern
}

// Parse - unmarshal recv syslog message to grpc event.
func (x *textParser) Parse(text string) (*pb.Event, error) {
	fields := strings.Fields(text)
	if arr, exist := x.patterns[len(fields)]; exist {
		for _, pattern := range arr {
			msg, err := pattern.unmarshal(fields...)
			if err == nil {
				return msg, nil
			}
			if err != ErrNotMatch {
				log.Debugf("parse err - %v", err)
			}
		}
	}

	return nil, fmt.Errorf("parse err - msg \"%s\" has unknown format ", text)
}

var (
	// pattern (event) keywords - parsing from yaml file
	eventKeyword = map[string]pb.EventType{
		"ignore":     pb.EventType_Unknown,
		"link_up":    pb.EventType_PortUp,
		"link_down":  pb.EventType_PortDown,
		"loopdetect": pb.EventType_PortLoopDetect,
	}
)

// textPattern - text message handler
type textPattern struct {
	eventType pb.EventType
	fields    []*textField
}

// newTextPattern - create new textPattern parser.
func newTextPattern(event, text string) (*textPattern, error) {
	p := &textPattern{
		fields: make([]*textField, 0),
	}
	if t, exist := eventKeyword[event]; exist {
		p.eventType = t
	} else {
		return nil, fmt.Errorf("unknown event type - %s", event)
	}
	fields := strings.Fields(text)
	for _, v := range fields {
		p.fields = append(p.fields, newTextField(v))
	}

	return p, nil
}

// unmarshal - try unmarshal recv data
func (p *textPattern) unmarshal(recv ...string) (*pb.Event, error) {
	if len(recv) != len(p.fields) {
		return nil, ErrNotMatch
	}
	result := &pb.Event{Type: p.eventType}
	for k, f := range p.fields {
		switch f.match(recv[k]) {
		case -1:
			return nil, ErrNotMatch
		case deviceAddr:
			{
				addr, err := parseDeviceAddr(recv[k])
				if err != nil {
					return nil, err
				}
				result.Host = addr
			}
		case devicePort:
			{
				port, err := parseDevicePort(recv[k])
				if err != nil {
					return nil, err
				}
				result.Port = port
			}
		case portSpeed:
			{
				speed, err := parsePortSpeed(recv[k])
				if err != nil {
					return nil, err
				}
				result.Speed = speed
			}
		case portDuplex:
			{
				dupx, err := parsePortDuplex(recv[k])
				if err != nil {
					return nil, err
				}
				result.Duplex = dupx
			}
		}
	}

	return result, nil
}

// parseDeviceAddr - aux ip address parse func.
func parseDeviceAddr(s string) (string, error) {
	if ip := net.ParseIP(s); ip != nil {
		return ip.String(), nil
	}
	return "", errors.New("ip address has invalid format")
}

// parseDevicePort - aux device port parse func.
func parseDevicePort(s string) (uint32, error) {
	nums := digitsOnly.FindAllString(s, -1)
	if len(nums) < 1 {
		return 0, errors.New("unknown port index format")
	}
	index, err := strconv.ParseUint(nums[len(nums)-1], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parsing port index err - %v", err)
	}

	return uint32(index), nil
}

// parsePortSpeed - aux port speed parse func.
func parsePortSpeed(s string) (pb.PortSpeed, error) {
	nums := digitsOnly.FindAllString(s, -1)
	if len(nums) < 1 {
		return 0, errors.New("unknown port speed format")
	}
	switch nums[0] {
	case "10":
		return pb.PortSpeed_Speed10Mb, nil
	case "100":
		return pb.PortSpeed_Speed100Mb, nil
	case "1000":
		return pb.PortSpeed_Speed1Gb, nil
	}

	return pb.PortSpeed_UnknownSpeed, errors.New("unknown port speed format")
}

// parsePortDuplex - aux port duplex parse func.
func parsePortDuplex(s string) (pb.PortDuplex, error) {
	if strings.Contains(strings.ToLower(s), "half") {
		return pb.PortDuplex_Half, nil
	}
	if strings.Contains(strings.ToLower(s), "full") {
		return pb.PortDuplex_Full, nil
	}

	return pb.PortDuplex_UnknownDuplex, errors.New("unknown port duplex format")
}

// avaliable data field types - must have equal elem count with fieldKeyword slice.
const (
	plainText = iota
	deviceAddr
	devicePort
	portSpeed
	portDuplex
)

var (
	// data type keywords - for pattern handle.
	fieldKeyword = []string{
		"$plain_text_do_not_use$",
		"$device_addr$",
		"$device_port$",
		"$port_speed$",
		"$port_duplex$",
	}

	// ErrNotMatch - infroms that text is not match with pattern.
	ErrNotMatch = errors.New("not match")

	// digitsOnly - aux regexp
	digitsOnly = regexp.MustCompile("[0-9]+")
)

// textField - basic text field of parser pattern.
type textField struct {
	dataType int
	text     string
}

// newTextField - create new text field for pattern.
func newTextField(text string) *textField {
	for k, v := range fieldKeyword {
		if text == v {
			return &textField{
				dataType: k,
			}
		}
	}

	return &textField{
		dataType: plainText,
		text:     text,
	}
}

// match - compare text field with current field.
func (f *textField) match(text string) int {
	if f.dataType != plainText {
		return f.dataType
	}
	if f.text == text {
		return 0
	}

	return -1
}
