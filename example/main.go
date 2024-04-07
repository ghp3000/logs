package main

import "github.com/ghp3000/logs"

func main() {
	logs.SetLevel(logs.LEVEL_ALL)
	logs.SetFormat(logs.FORMAT_MICROSECNDS | logs.FORMAT_SHORTFILENAME)
	str := "hello %s"
	logs.Debug(str, "word")
	log, _ := logs.NewLogger().SetConsole(false).SetFormat(logs.FORMAT_SHORTFILENAME|logs.FORMAT_MICROSECNDS).
		SetRollingFile(`D:\cfoldTest\`, `ghp-t.txt`, 1, logs.GB)
	log.Debug(str, "word")
}
