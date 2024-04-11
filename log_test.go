package logs

import "testing"

func BenchmarkSerialLog(b *testing.B) {
	b.StopTimer()
	//console := NewConsoleLog(LevelAll, DefaultTimeFormatShort, DefaultLogFormat, "D:/project/")
	file, err := NewFileLog(LevelAll, DefaultTimeFormatShort, DefaultLogFormat, "")
	if err != nil {
		panic(err)
	}
	log := New(2, file)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		log.Debug("%d ,>>>aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", i)
	}
}
