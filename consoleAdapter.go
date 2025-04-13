package logs

import (
	"bytes"
	"fmt"
	"strconv"
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
			timeBuf:    make([]byte, 0, 64),
		},
	}
}
func (c *ConsoleLog) Write(item *Item) {
	//defer c.clean(item)
	defer item.Put()
	if c.level > item.Level {
		return
	}
	buf := bytes.NewBuffer(nil)
	fmt.Print(c.formatItem(buf, item).String())
	buf.Reset()
}
func (c *ConsoleLog) formatItem(buf *bytes.Buffer, item *Item) *bytes.Buffer {
	//buf.WriteString(item.Time.Format(c.timeFormat))
	c.timeBuf = c.timeBuf[:0]
	buf.Write(item.Time.AppendFormat(c.timeBuf, c.timeFormat))
	buf.WriteString(" [")
	buf.WriteString(item.Level.Name())
	buf.WriteString("] ")
	if c.trim {
		buf.WriteString(strings.TrimPrefix(item.File, c.trimPath))
	} else {
		buf.WriteString(item.File)
	}
	//utils\example\example.go:60
	buf.WriteString(":")
	buf.WriteString(strconv.Itoa(item.Line))
	buf.WriteString(" ")
	buf.WriteString(item.Content)
	buf.WriteByte('\n')
	return buf
}
func (c *ConsoleLog) Close() {
	c.level = LevelOff
}
