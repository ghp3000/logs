package logs

import (
	"fmt"
	"strings"
)

type ConsoleLog struct {
	BaseAdapter
}

func NewConsoleLog(level LEVEL, timeFormat, format string, trimPath string) Adapter {
	return &ConsoleLog{
		BaseAdapter: BaseAdapter{
			level:      level,
			timeFormat: timeFormat,
			format:     format,
			trimPath:   trimPath,
			trim:       len(trimPath) != 0,
			name:       "console",
		},
	}
}
func (c *ConsoleLog) Write(item *Item) {
	if c.level > item.Level {
		return
	}
	if c.trim {
		fmt.Printf(c.format, item.Time.Format(c.timeFormat), item.Level.Name(), strings.TrimLeft(item.File, c.trimPath), item.Line, item.Content)
	} else {
		fmt.Printf(c.format, item.Time.Format(c.timeFormat), item.Level.Name(), item.File, item.Line, item.Content)
	}
}
func (c *ConsoleLog) Close() {
	c.level = LevelOff
}
