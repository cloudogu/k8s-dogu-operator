package logging

import "github.com/go-logr/logr"

type LogSink interface {
	logr.LogSink
}
