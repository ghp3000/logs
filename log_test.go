package logs

import "testing"

func BenchmarkSerialLog(b *testing.B) {
	b.StopTimer()
	//console := NewConsoleLog(LevelAll, DefaultTimeFormatShort, DefaultLogFormat, "D:/project/")
	cfg := &FileConfig{
		Dir:         "D:\\cfoldTest",
		FileName:    "test.log",
		FileMaxSize: 100 * MB,
		FileMaxNum:  5,
		RollType:    RollingFile,
		Gzip:        false,
	}
	file, err := NewFileLog(LevelAll, cfg, 100, "D:/project/")
	if err != nil {
		panic(err)
	}
	log := New(2, file)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		log.Debug("%d ,>>>aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", i)
	}
}
