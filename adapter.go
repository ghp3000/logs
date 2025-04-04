package logs

import (
	"sync/atomic"
	"time"
)

type Adapter interface {
	Name() string
	Write(item *Item)
	Level() LEVEL
	SetLevel(level LEVEL)
	SetFormat(s string)     //%s [%s] %s:%d %s
	SetTimeFormat(s string) // 比如:2006-01-02 15:04:05
	Close()
}

const (
	DefaultLogFormat       = "%s [%s] %s:%d %s\n"
	DefaultTimeFormatLong  = "2006-01-02 15:04:05.000"
	DefaultTimeFormatShort = "01-02 15:04:05.000"
)

type BaseAdapter struct {
	level      LEVEL
	timeFormat string
	format     string
	trimPath   string
	trim       bool
	name       string
}

func (c *BaseAdapter) SetTimeFormat(s string) {
	c.timeFormat = s
}
func (c *BaseAdapter) SetFormat(s string) {
	c.format = s
}
func (c *BaseAdapter) Name() string {
	return c.name
}
func (c *BaseAdapter) Write(item *Item) {
	if c.level > item.Level {
		return
	}
}
func (c *BaseAdapter) Level() LEVEL {
	return c.level
}
func (c *BaseAdapter) SetLevel(level LEVEL) {
	c.level = level
}
func (c *BaseAdapter) Close() {
	c.level = LevelOff
}
func (c *BaseAdapter) clean(item *Item) {
	if atomic.AddInt32(&item.count, -1) == 0 {
		item.File = ""
		item.Line = 0
		item.Content = ""
		item.Time = time.Time{}
		item.pool.Put(item)
	}
}
