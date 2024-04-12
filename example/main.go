package main

import (
	"github.com/ghp3000/logs"
)

func main() {
	//logs.SetLevel(logs.LEVEL_ALL)
	//logs.SetFormat(logs.FORMAT_MICROSECNDS | logs.FORMAT_SHORTFILENAME)
	//str := "hello %s"
	//logs.Debug(str, "word")
	//log, _ := logs.NewLogger().SetConsole(true).SetFormat(logs.FORMAT_SHORTFILENAME|logs.FORMAT_MICROSECNDS).
	//	SetRollingFile(`D:\cfoldTest\`, `ghp-t.txt`, 1, logs.GB)
	//log.Debug(str, "word")
	//console := logs.NewConsoleLog(logs.LevelAll, logs.DefaultTimeFormatShort, logs.DefaultLogFormat, "D:/project/")
	cfg := &logs.FileConfig{
		Dir:         "D:\\cfoldTest",
		FileName:    "test.log",
		FileMaxSize: 100 * logs.MB,
		FileMaxNum:  5,
		RollType:    logs.RollingFile,
		Gzip:        false,
	}
	file, err := logs.NewFileLog(logs.LevelAll, cfg, 100, "D:/project/")
	if err != nil {
		panic(err)
	}
	log := logs.New(2, file)
	for i := 0; i < 10000000; i++ {
		log.Debug("hello %s", "word,xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		//time.Sleep(time.Microsecond)
	}

	//time.Sleep(time.Second * 30)
}
