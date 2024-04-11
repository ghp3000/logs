package logs

import (
	"fmt"
	"github.com/donnie4w/gofer/hashmap"
	"os"
	"runtime"
	"strings"
	"time"
)

var mp = hashmap.NewLimitMap[any, runtime.Frame](1 << 13)

type LEVEL int8

func (v LEVEL) Name() string {
	switch v {
	case LevelAll:
		return "All"
	case LevelDebug:
		return "Debug"
	case LevelInfo:
		return "Info"
	case LevelError:
		return "Error"
	case LevelWarn:
		return "Warn"
	case LevelFatal:
		return "Fatal"
	default:
		return "Off"
	}
}

const (
	LevelAll LEVEL = iota //ALL<DEBUG<INFO<WARN<error<fatal<OFF
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelOff
)

type Item struct {
	Level   LEVEL
	Time    time.Time
	File    string
	Line    int
	Content string
}

type GLogger struct {
	level     LEVEL //基础日志级别.
	callDepth int
	adapters  map[string]Adapter
}

func New(callDepth int, adapter ...Adapter) *GLogger {
	ad := &GLogger{
		level:     LevelOff,
		callDepth: callDepth,
		adapters:  make(map[string]Adapter),
	}
	for _, a := range adapter {
		ad.AddAdapter(a)
	}
	return ad
}
func (l *GLogger) Debug(f interface{}, v ...interface{}) {
	l.print(LevelDebug, f, v...)
}
func (l *GLogger) Info(f interface{}, v ...interface{}) {
	l.print(LevelInfo, f, v...)
}
func (l *GLogger) Warn(f interface{}, v ...interface{}) {
	l.print(LevelWarn, f, v...)
}
func (l *GLogger) Error(f interface{}, v ...interface{}) {
	l.print(LevelError, f, v...)
}
func (l *GLogger) Fatal(f interface{}, v ...interface{}) {
	l.print(LevelFatal, f, v...)
}

func (l *GLogger) Close(f interface{}, v ...interface{}) {
	for _, adapter := range l.adapters {
		adapter.Close()
	}
	l.level = LevelOff
}

func (l *GLogger) print(level LEVEL, f interface{}, v ...interface{}) {
	if l.level > level {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(l.callDepth+1, pcs[:])
	var frame runtime.Frame
	var ok bool
	if frame, ok = mp.Get(pcs); !ok {
		frame, _ = runtime.CallersFrames([]uintptr{pcs[0]}).Next()
		mp.Put(pcs, frame)
	}
	item := &Item{
		Level:   level,
		Time:    time.Now(),
		File:    frame.File,
		Line:    frame.Line,
		Content: l.formatPattern(f, v...),
	}
	for _, adapter := range l.adapters {
		adapter.Write(item)
	}
}
func (l *GLogger) formatPattern(f interface{}, v ...interface{}) string {

	fstr := l.format(f, v...)
	if len(v) > 0 {
		fstr = fmt.Sprintf(fstr, v...)
	}
	return fstr
}
func (l *GLogger) format(f interface{}, v ...interface{}) string {
	var msg string
	switch f.(type) {
	case string:
		msg = f.(string)
		if len(v) == 0 {
			return msg
		}
		if !strings.Contains(msg, "%") {
			msg += strings.Repeat(" %v", len(v))
		}
	default:
		msg = fmt.Sprint(f)
		if len(v) == 0 {
			return msg
		}
		msg += strings.Repeat(" %v", len(v))
	}
	return msg
}

// calcMaxLevel 遍历Adapter,取最详细的日志级别赋值给当前日志级别
func (l *GLogger) calcMaxLevel() {
	l.level = LevelOff
	for _, adapter := range l.adapters {
		if adapter.Level() < l.level {
			l.level = adapter.Level()
		}
	}
}

func (l *GLogger) Level() LEVEL {
	return l.level
}
func (l *GLogger) AddAdapter(adapter Adapter) {
	if adapter == nil {
		return
	}
	l.adapters[adapter.Name()] = adapter
	l.calcMaxLevel()
}
func (l *GLogger) RemoveAdapter(name string) {
	delete(l.adapters, name)
	l.calcMaxLevel()
}
func (l *GLogger) Adapters() (ret []string) {
	for s, _ := range l.adapters {
		ret = append(ret, s)
	}
	return
}
func (l *GLogger) Adapter(name string) Adapter {
	return l.adapters[name]
}

func mkdirDir(dir string) (e error) {
	_, er := os.Stat(dir)
	b := er == nil || os.IsExist(er)
	if !b {
		if err := os.MkdirAll(dir, 0666); err != nil {
			if os.IsPermission(err) {
				e = err
			}
		}
	}
	return
}

func isFileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func catchError() {
	if err := recover(); err != nil {
		//Fatal(string(debug.Stack()))
	}
}
