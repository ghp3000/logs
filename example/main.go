package main

import "github.com/ghp3000/logs"

func main() {
	logs.SetLevel(logs.LEVEL_ALL)
	logs.SetFormat(logs.FORMAT_MICROSECNDS | logs.FORMAT_SHORTFILENAME)
	str := "hello %s"
	logs.Debug(str)
}
