package logs

import (
	"fmt"
	"github.com/donnie4w/gofer/hashmap"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
	count   int32
	pool    *sync.Pool
}

func (i *Item) Put() {
	if atomic.AddInt32(&i.count, -1) < 1 {
		//i.File = ""
		//i.Line = 0
		//i.Content = ""
		//i.Time = time.Time{}
		i.pool.Put(i)
	}
}

type GLogger struct {
	level     LEVEL //基础日志级别.
	callDepth int
	adapters  map[string]Adapter
	pool      sync.Pool
}

func New(callDepth int, adapter ...Adapter) *GLogger {
	ad := &GLogger{
		level:     LevelOff,
		callDepth: callDepth,
		adapters:  make(map[string]Adapter),
		pool: sync.Pool{
			New: func() any {
				return new(Item)
			},
		},
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

func (l *GLogger) Close() {
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
	item := l.pool.Get().(*Item)
	item.Level = level
	item.Time = time.Now()
	item.File = frame.File
	item.Line = frame.Line
	item.Content = l.formatPattern(f, v...)
	item.count = 0
	item.pool = &l.pool
	for _, adapter := range l.adapters {
		atomic.AddInt32(&item.count, 1)
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

func (l *GLogger) GetLevel() LEVEL {
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
func (l *GLogger) GetAdapters() (ret []string) {
	for s := range l.adapters {
		ret = append(ret, s)
	}
	return
}
func (l *GLogger) GetAdapter(name string) Adapter {
	return l.adapters[name]
}
func (l *GLogger) SetCallDepth(callDepth int) {
	l.callDepth = callDepth
}
func catchError() {
	if err := recover(); err != nil {
		//Fatal(string(debug.Stack()))
	}
}
