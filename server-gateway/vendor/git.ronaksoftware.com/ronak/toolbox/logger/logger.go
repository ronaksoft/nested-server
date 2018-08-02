package log

const (
    _ = iota
    LEVEL_DEBUG             // 0x01
    LEVEL_INFO              // 0x02
    LEVEL_ERROR             // 0x03
    LEVEL_FATAL             // 0x03
)

type Logger struct {
    l int
    b LoggerBackend
}

func NewLogger(level int, b LoggerBackend) *Logger {
    l := new(Logger)
    l.b = b
    l.SetLevel(level)
    return l
}

func NewTerminalLogger(level int) *Logger {
    l := new(Logger)
    l.b = new(TerminalLogger)
    l.SetLevel(level)
    return l
}

func (l *Logger) SetLevel(level int) {
    switch level {
    case LEVEL_DEBUG, LEVEL_ERROR, LEVEL_FATAL, LEVEL_INFO:
        l.l = level
    default:
        l.l = LEVEL_ERROR
    }
}

func (l *Logger) Debug(identifier, text string, args ... interface{}) {
    if l.l > LEVEL_DEBUG {
        return
    }
    l.b.WriteDebugLog(identifier, text, args ...)
}

func (l *Logger) Info(identifier, text string, args ... interface{}) {
    if l.l > LEVEL_INFO {
        return
    }
    l.b.WriteInfoLog(identifier, text, args ...)
}

func (l *Logger) Error(errorIdentifier, errText string, args ...interface{}) {
    if l.l > LEVEL_ERROR {
        return
    }
    l.b.WriteErrorLog(errorIdentifier, errText, args ...)
}

func (l *Logger) Fatal(errorIdentifier, errText string, args ...interface{}) {
    if l.l > LEVEL_FATAL {
        return
    }
    l.b.WriteFatalLog(errorIdentifier, errText, args ...)
}
