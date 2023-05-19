package log

// Logger represents a logger
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// NoopLogger is a log doesn't do anything
type NoopLogger struct{}

func (l *NoopLogger) Debugf(format string, args ...interface{}) {}
func (l *NoopLogger) Infof(format string, args ...interface{})  {}
func (l *NoopLogger) Warnf(format string, args ...interface{})  {}
func (l *NoopLogger) Errorf(format string, args ...interface{}) {}
