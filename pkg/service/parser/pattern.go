package parser

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
)

// ErrDataParse - вспомогательный тип данных
// для проблем обработки входящего сообщения.
type ErrDataParse struct {
	Message string
}

// Error - реализация итерфейса Error.
func (e *ErrDataParse) Error() string {
	return e.Message
}

var (
	// ErrNotMatch - указанный текст не совпадает с текущим шаблоном.
	ErrNotMatch = errors.New("not match")

	// вспомательное регулярное выражение для обработки текстовых полей.
	digitsOnly = regexp.MustCompile("[0-9]+")

	// Вспомогательные значения типа события шаблона.
	eventKeyword = map[string]pb.EventType{
		"ignore":     pb.EventType_Unknown,
		"link_up":    pb.EventType_PortUp,
		"link_down":  pb.EventType_PortDown,
		"loopdetect": pb.EventType_PortLoopDetect,
	}
)

// textPattern - шаблон обработки текстовых сообщений.
type textPattern struct {
	eventType pb.EventType
	fields    []*textField
}

// newTextPattern - создать новый экземпляр обработчика на базе шаблона.
func newTextPattern(event, text string) (*textPattern, error) {
	p := &textPattern{
		fields: make([]*textField, 0),
	}
	if t, exist := eventKeyword[event]; exist {
		p.eventType = t
	} else {
		return nil, fmt.Errorf("new text pattern - unknown event type - %s", event)
	}
	fields := strings.Fields(text)
	for _, v := range fields {
		p.fields = append(p.fields, newTextField(v))
	}

	return p, nil
}

// unmarshal - преобразовать поля текстового сообщения в формат grpc.
// Возвращает ErrNotMatch - если текстовое сообщение не совпадает с шаблоном,
// или ошибку уровня обработки
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
					return nil, &ErrDataParse{Message: fmt.Sprintf("device address parse err - %v", err)}
				}
				result.Host = addr
			}
		case devicePort:
			{
				port, err := parseDevicePort(recv[k])
				if err != nil {
					return nil, &ErrDataParse{Message: fmt.Sprintf("device port parse err - %v", err)}
				}
				result.Port = port
			}
		case portSpeed:
			{
				speed, err := parsePortSpeed(recv[k])
				if err != nil {
					return nil, &ErrDataParse{Message: fmt.Sprintf("device port speed parse err - %v", err)}
				}
				result.Speed = speed
			}
		case portDuplex:
			{
				dupx, err := parsePortDuplex(recv[k])
				if err != nil {
					return nil, &ErrDataParse{Message: fmt.Sprintf("device port duplex parse err - %v", err)}
				}
				result.Duplex = dupx
			}
		}
	}

	return result, nil
}

// parseDeviceAddr - вспомогательная функция обработки данных.
func parseDeviceAddr(s string) (string, error) {
	if ip := net.ParseIP(s); ip != nil {
		return ip.String(), nil
	}
	return "", errors.New("ip address has invalid format")
}

// parseDevicePort - вспомогательная функция обработки данных.
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

// parsePortSpeed - вспомогательная функция обработки данных.
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

// parsePortDuplex - вспомогательная функция обработки данных.
func parsePortDuplex(s string) (pb.PortDuplex, error) {
	if strings.Contains(strings.ToLower(s), "half") {
		return pb.PortDuplex_Half, nil
	}
	if strings.Contains(strings.ToLower(s), "full") {
		return pb.PortDuplex_Full, nil
	}

	return pb.PortDuplex_UnknownDuplex, errors.New("unknown port duplex format")
}
