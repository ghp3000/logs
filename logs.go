package logs

var defaultLogger *GLogger = New(2, NewConsoleLog(LevelAll, DefaultTimeFormatShort, DefaultLogFormat, ""))

func Debug(f interface{}, v ...interface{}) {
	defaultLogger.print(LevelDebug, f, v...)
}
func Info(f interface{}, v ...interface{}) {
	defaultLogger.print(LevelInfo, f, v...)
}
func Warn(f interface{}, v ...interface{}) {
	defaultLogger.print(LevelWarn, f, v...)
}
func Error(f interface{}, v ...interface{}) {
	defaultLogger.print(LevelError, f, v...)
}
func Fatal(f interface{}, v ...interface{}) {
	defaultLogger.print(LevelFatal, f, v...)
}
func GetLevel() LEVEL {
	return defaultLogger.GetLevel()
}
func AddAdapter(adapter Adapter) {
	defaultLogger.AddAdapter(adapter)
}
func RemoveAdapter(name string) {
	defaultLogger.RemoveAdapter(name)
}
func GetAdapters() (ret []string) {
	return defaultLogger.GetAdapters()
}
func GetAdapter(name string) Adapter {
	return defaultLogger.GetAdapter(name)
}
func SetCallDepth(callDepth int) {
	defaultLogger.SetCallDepth(callDepth)
}
func Close() {
	defaultLogger.Close()
}
