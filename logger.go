package httpclient

import (
	"fmt"
	"io"
)

type LoggerAdapter struct {
	Writer io.Writer
}

func (l *LoggerAdapter) Debugf(format string, v ...interface{}) {
	l.logf("DEBUG: "+format, v...)
}

func (l *LoggerAdapter) Infof(format string, v ...interface{}) {
	l.logf("INFO: "+format, v...)
}

func (l *LoggerAdapter) Warnf(format string, v ...interface{}) {
	l.logf("WARN: "+format, v...)
}

func (l *LoggerAdapter) Errorf(format string, v ...interface{}) {
	l.logf("ERROR: "+format, v...)
}

func (l *LoggerAdapter) logf(format string, v ...interface{}) {
	fmt.Fprintf(l.Writer, format+"\n", v...)
}
