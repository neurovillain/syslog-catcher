package parser

import (
	"errors"
	"fmt"
	"strings"

	pb "github.com/neurovillain/syslog-catcher/pkg/api/proto"
	log "github.com/sirupsen/logrus"
)

const (
	// разделитель данных "тип сообщения"-"тело сообщения"
	patternTypeDelim = " ~ "
)

// Parser - интерфейс обработчика текстовых данных.
type Parser interface {
	// Parse - преобразовать сообщение в формат события GRPC.
	Parse(string) (*pb.Event, error)
}

// NewParser - cоздать новый экземпляр Parser.
// входные данные - набор шаблонов для обработки данных.
func NewParser(patterns []string) (Parser, error) {
	if len(patterns) == 0 {
		return nil, errors.New("no patterns for parser are provided")
	}
	result := &textParser{
		patterns: make(map[int][]*textPattern),
	}
	for _, v := range patterns {
		args := strings.SplitN(v, patternTypeDelim, 2)
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
	log.Debugf("defined %d text parser patterns", len(patterns))

	return result, nil
}

// textParser - реализация интерфейса Parser.
type textParser struct {
	patterns map[int][]*textPattern
}

// Parse - преобразовать сообщение в формат события GRPC.
func (x *textParser) Parse(text string) (*pb.Event, error) {
	fields := strings.Fields(text)
	if arr, exist := x.patterns[len(fields)]; exist {
		for _, pattern := range arr {
			msg, err := pattern.unmarshal(fields...)
			if err == nil {
				return msg, nil
			}
			if err != ErrNotMatch {
				log.Warnf("parse err - msg %s - %v", text, err)
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("parse err - msg \"%s\" has unknown format ", text)
}
