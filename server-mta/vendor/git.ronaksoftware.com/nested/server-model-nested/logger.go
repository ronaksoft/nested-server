package nested

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"log"
	"time"
)

const (
	LOG_LEVEL_FUNCTION int = 0x01
	LOG_LEVEL_QUERY    int = 0x02
	LOG_LEVEL_ERROR    int = 0x03
)

type Logger struct {
	enabledLevels map[int]bool
	chLogs        chan interface{}
	chShutdown    chan bool
	topFunction   FunctionLog
	fDepth        int
}

type FunctionLog struct {
	FunctionName string    `bson:"func_name"`
	CallTime     time.Time `bson:"call_time"`
	Duration     int       `bson:"duration"` // milli-seconds

	FunctionArgs []interface{} `bson:"func_args"`
	SubFunctions []FunctionLog `bson:"sub_functions"`
}
type QueryLog struct {
	FunctionName string      `bson:"func_name"`
	CallTime     time.Time   `bson:"call_time"`
	Explain      interface{} `bson:"explain"`
	Query        interface{} `bson:"query"`
}
type ErrorLog struct {
	FunctionName string        `bson:"func_name"`
	ErrorTime    time.Time     `bson:"err_time"`
	ErrorText    string        `bson:"err_text"`
	Args         []interface{} `bson:"args"`
}

func NewLogger() *Logger {
	l := new(Logger)
	l.enabledLevels = make(map[int]bool)
	l.chLogs = make(chan interface{}, 10)
	l.chShutdown = make(chan bool)
	go l.thread()
	return l
}

// Internal Functions
func (l *Logger) thread() {
	logItems := make([]interface{}, 0, 12)
	for {
		select {
		case in := <-l.chLogs:
			logItems = append(logItems, in)
			if len(logItems) >= 10 {
				l.insertDocs(logItems...)
				logItems = logItems[:0]
			}
		case <-time.Tick(10 * time.Second):
			if len(logItems) > 0 {
				l.insertDocs(logItems...)
				logItems = logItems[:0]
			}
		case <-l.chShutdown:
			if len(logItems) > 0 {
				l.insertDocs(logItems...)
				logItems = logItems[:0]
			}
			break
		}
	}
}
func (l *Logger) insertDocs(docs ...interface{}) {
	var constructor, funcName string
	mongoDocs := make([]interface{}, 0, len(docs))
	for _, doc := range docs {
		switch x := doc.(type) {
		case FunctionLog:
			constructor = "FUNC"
			funcName = x.FunctionName
		case QueryLog:
			constructor = "QUERY"
			funcName = x.FunctionName
		case ErrorLog:
			constructor = "ERROR"
			funcName = x.FunctionName
		default:
			log.Println(x)
		}
		mongoDocs = append(mongoDocs, M{
			"_id":         bson.NewObjectId(),
			"constructor": constructor,
			"func_name":   funcName,
			"data":        doc,
		})
	}
	if err := _MongoDB.C(COLLECTION_LOGS).Insert(mongoDocs...); err != nil {
		log.Println("Logger::insertDocs::Error::", err.Error())
	}
}
func (l *Logger) levelEnabled(level int) bool {
	if v, ok := l.enabledLevels[level]; ok && v {
		return true
	}
	return false
}

// Exposed Functions
func (l *Logger) EnableLevel(levels ...int) {
	for _, level := range levels {
		l.enabledLevels[level] = true
	}
}
func (l *Logger) FunctionStarted(funcName string, args ...interface{}) {
	if !l.levelEnabled(LOG_LEVEL_FUNCTION) {
		return
	}
	if l.fDepth == 0 {
		l.topFunction = FunctionLog{
			FunctionName: funcName,
			CallTime:     time.Now(),
			FunctionArgs: args,
			SubFunctions: []FunctionLog{},
		}
	} else {
		l.topFunction.SubFunctions = append(l.topFunction.SubFunctions, FunctionLog{
			FunctionName: funcName,
			CallTime:     time.Now(),
			FunctionArgs: args,
		})
	}
	l.fDepth++
}
func (l *Logger) FunctionFinished(funcName string) {
	if !l.levelEnabled(LOG_LEVEL_FUNCTION) {
		return
	}
	l.fDepth--
	if l.fDepth == 0 {
		l.topFunction.Duration = int(time.Now().Sub(l.topFunction.CallTime).Nanoseconds() / 1e6)
		l.chLogs <- l.topFunction
	}
}
func (l *Logger) ExplainQuery(funcName string, Q *mgo.Query) {
	if !l.levelEnabled(LOG_LEVEL_QUERY) {
		return
	}
	r := M{}
	Q.Explain(r)
	qLog := QueryLog{
		CallTime:     time.Now(),
		FunctionName: funcName,
		Explain:      r["executionStats"],
	}
	l.chLogs <- qLog
}
func (l *Logger) ExplainPipe(funcName string, Q *mgo.Pipe) {
	if !l.levelEnabled(LOG_LEVEL_QUERY) {
		return
	}
	r := M{}
	Q.Explain(r)
	qLog := QueryLog{
		CallTime:     time.Now(),
		FunctionName: funcName,
		Explain:      r["executionStats"],
	}
	l.chLogs <- qLog
}
func (l *Logger) Error(funcName, errText string, args ...interface{}) {
	log.Println(funcName, "::", errText, args)
	eLog := ErrorLog{
		ErrorTime:    time.Now(),
		FunctionName: funcName,
		ErrorText:    errText,
		Args:         args,
	}
	l.chLogs <- eLog
}
func (l *Logger) Shutdown() {
	l.chShutdown <- true
}
