package log

import (
    "fmt"
    "os"
)


type LoggerBackend interface {
    WriteDebugLog(identifier, text string, args ... interface{})
    WriteInfoLog(identifier, text string, args ...interface{})
    WriteErrorLog(errIdentifier, errText string, args ... interface{})
    WriteFatalLog(errIdentifier, errText string, arg ... interface{})
}


// TerminalLogger implements LoggerBackend interface and use terminal for the logs
type TerminalLogger struct {}

func (b *TerminalLogger) WriteDebugLog(identifier, text string, args ... interface{}) {
    inputText := fmt.Sprintf("%s::%s", identifier, text)
    fmt.Println("DEBUG::", inputText)
    if len(args) > 0 {
        fmt.Println("--> ", args)
    }
}

func (b *TerminalLogger) WriteInfoLog(identifier, text string, args ... interface{}) {
    inputText := fmt.Sprintf("%s::%s", identifier, text)
    fmt.Println("INFO::", inputText)
    if len(args) > 0 {
        fmt.Println("--> ", args)
    }
}

func (b *TerminalLogger) WriteErrorLog(errIdentifier, errText string, args ... interface{}) {
    inputText := fmt.Sprintf("%s::%s", errIdentifier, errText)
    fmt.Println("ERROR::", inputText)
    if len(args) > 0 {
        fmt.Println("--> ", args)
    }
}

func (b *TerminalLogger) WriteFatalLog(errIdentifier, errText string, args ... interface{}) {
    inputText := fmt.Sprintf("%s::%s", errIdentifier, errText)
    fmt.Println("FATAL::", inputText)
    if len(args) > 0 {
        fmt.Println("--> ", args)
    }
    os.Exit(1)
}
