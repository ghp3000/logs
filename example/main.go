package main

import (
	"fmt"
	"github.com/ghp3000/logs"
	"time"
)

func main() {
	//logs.SetLevel(logs.LEVEL_ALL)
	//logs.SetFormat(logs.FORMAT_MICROSECNDS | logs.FORMAT_SHORTFILENAME)
	//str := "hello %s"
	//logs.Debug(str, "word")
	//log, _ := logs.NewLogger().SetConsole(true).SetFormat(logs.FORMAT_SHORTFILENAME|logs.FORMAT_MICROSECNDS).
	//	SetRollingFile(`D:\cfoldTest\`, `ghp-t.txt`, 1, logs.GB)
	//log.Debug(str, "word")
	fmt.Println()
	console := logs.NewConsoleLog(logs.LevelAll, logs.DefaultTimeFormatShort, logs.DefaultLogFormat, "D:/project/")
	file, err := logs.NewFileLog(logs.LevelAll, logs.DefaultTimeFormatShort, logs.DefaultLogFormat, "D:/project/")
	if err != nil {
		panic(err)
	}
	log := logs.New(2, console, file)
	log.Debug("hello %s", "word")
	time.Sleep(time.Second * 30)
}
