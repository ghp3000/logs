package logs

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type TimeMode uint8
type rollingType uint8

const (
	ModeHour  TimeMode = 1
	ModeDay   TimeMode = 2
	ModeMonth TimeMode = 3

	RollingDaily rollingType = 0 //按日期分割文件
	RollingFile  rollingType = 1 //按文件大小分割文件

	dateformatDay   = "20060102"
	dateformatHour  = "2006010215"
	dateformatMonth = "200601"
)

const (
	_  = iota
	KB = 1 << (iota * 10)
	MB
	GB
	TB
)

type FileLog struct {
	BaseAdapter
	cache   chan *Item
	fs      *fileStore
	sync    bool
	_rwLock sync.RWMutex
	cancel  chan struct{}
}

func NewFileLog(level LEVEL, timeFormat, format string, trimPath string) (Adapter, error) {
	f := &FileLog{
		BaseAdapter: BaseAdapter{
			level:      level,
			timeFormat: timeFormat,
			format:     format,
			trimPath:   trimPath,
			trim:       len(trimPath) != 0,
			name:       "file",
		},
		cache: make(chan *Item, 100),
		sync:  false,
		fs:    new(fileStore),
	}
	f.fs.fileDir = "d:\\cfoldTest"
	f.fs.fileName = "test.log"
	f.fs.maxSize = 10 * MB
	f.fs.fileSize = MB
	f.fs.rollType = RollingFile
	f.fs.mode = ModeDay
	f.fs.gzip = false
	if err := f.fs.openFileHandler(); err != nil {
		return nil, err
	}
	f.cancel = make(chan struct{})
	if !f.sync {
		go f.loop()
	}
	return f, nil
}
func (c *FileLog) loop() {
	defer catchError()
	var item *Item
	defer c.fs.close()
	buf := bytes.NewBuffer(nil)
	for {
		select {
		case item = <-c.cache:
			c.write(buf, item)
		case <-c.cancel:
			close(c.cancel)
			close(c.cache)
			return
		}
	}
}
func (c *FileLog) Write(item *Item) {
	if c.level > item.Level {
		return
	}
	if c.sync {
		buf := bytes.NewBuffer(nil)
		c.write(buf, item)
	} else {
		c.cache <- item
	}
}
func (c *FileLog) write(buf *bytes.Buffer, item *Item) (bakfn string, err error) {
	if item == nil {
		return
	}
	if c.fs.isFileWell {
		var openFileErr error
		if c.fs.isMustBackUp() {
			bakfn, err, openFileErr = c.backUp()
		}
		if openFileErr == nil {
			c._rwLock.RLock()
			defer c._rwLock.RUnlock()
			c.fs.write2file(c.formatItem(buf, item).Bytes())
			buf.Reset()
			return
		}
	}
	return
}
func (c *FileLog) backUp() (bakfn string, err, openFileErr error) {
	c._rwLock.Lock()
	defer c._rwLock.Unlock()
	if !c.fs.isMustBackUp() {
		return
	}
	err = c.fs.close()
	if err != nil {
		return
	}
	bakfn, err = c.fs.rename()
	if err != nil {

		return
	}
	openFileErr = c.fs.openFileHandler()
	if openFileErr != nil {

	}
	return
}
func (c *FileLog) Close() {
	defer catchError()
	c.cancel <- struct{}{}
	c.level = LevelOff
}
func (c *FileLog) formatItem(buf *bytes.Buffer, item *Item) *bytes.Buffer {
	buf.WriteString(item.Time.Format(c.timeFormat))
	buf.WriteString(" [")
	buf.WriteString(item.Level.Name())
	buf.WriteString("] ")
	if c.trim {
		buf.WriteString(strings.TrimLeft(item.File, c.trimPath))
	} else {
		buf.WriteString(item.File)
	}
	buf.WriteString(":[")
	buf.WriteString(strconv.Itoa(item.Line))
	buf.WriteString("]")
	buf.WriteString(item.Content)
	buf.WriteByte('\n')
	return buf
}

type fileStore struct {
	fileDir     string      //日志文件所在的目录
	fileName    string      //日志文件名
	maxSize     int64       //日志文件的最大大小
	fileSize    int64       //单个日志文件的大小
	rollType    rollingType //日志的滚动模式,按日期或按文件大小
	tomorSecond int64       //日志文件的下一天的时间戳
	isFileWell  bool        //文件是否正常
	maxFileNum  int         //日志文件最大数量
	mode        TimeMode    //日志文件滚动模式
	gzip        bool        //是否开启gzip

	fileHandler *os.File
	lock        sync.Mutex
}

func (t *fileStore) openFileHandler() (e error) {
	if t.fileDir == "" || t.fileName == "" {
		e = errors.New("log filePath is null or error")
		return
	}
	e = mkdirDir(t.fileDir)
	if e != nil {
		t.isFileWell = false
		return
	}
	fname := fmt.Sprint(t.fileDir, "/", t.fileName)
	t.fileHandler, e = os.OpenFile(fname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if e != nil {
		t.isFileWell = false
		return
	}
	t.isFileWell = true
	t.tomorSecond = tomorSecond(t.mode)
	if fs, err := t.fileHandler.Stat(); err == nil {
		t.fileSize = fs.Size()
	} else {
		e = err
	}
	return
}
func (t *fileStore) write2file(bs []byte) (n int, e error) {
	defer catchError()
	if bs != nil {
		if n, e = t.fileHandler.Write(bs); e == nil {
			atomic.AddInt64(&t.fileSize, int64(n))
		}
	}
	return
}
func (t *fileStore) isMustBackUp() bool {
	switch t.rollType {
	case RollingDaily:
		if time.Now().Unix() >= t.tomorSecond {
			return true
		}
	case RollingFile:
		return t.fileSize > 0 && t.fileSize >= t.maxSize
	}
	return false
}
func (t *fileStore) rename() (backupFileName string, err error) {
	if t.rollType == RollingDaily {
		backupFileName = t.getBackupDailyFileName(t.fileDir, t.fileName, t.mode, t.gzip)
	} else {
		backupFileName, err = t.getBackupRollFileName(t.fileDir, t.fileName, t.gzip)
	}
	if backupFileName != "" && err == nil {
		oldPath := fmt.Sprint(t.fileDir, "/", t.fileName)
		newPath := fmt.Sprint(t.fileDir, "/", backupFileName)
		err = os.Rename(oldPath, newPath)
		go func() {
			if err == nil && t.gzip {
				if err = lgzip(fmt.Sprint(newPath, ".gz"), backupFileName, newPath); err == nil {
					os.Remove(newPath)
				}
			}
			if err == nil && t.rollType == RollingFile && t.maxFileNum > 0 {
				t.rmOverCountFile(t.fileDir, backupFileName, t.maxFileNum, t.gzip)
			}
		}()
	}
	return
}
func (t *fileStore) rmOverCountFile(dir, backupfileName string, maxFileNum int, isGzip bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	f, err := os.Open(dir)
	if err != nil {
		return
	}
	dirs, _ := f.ReadDir(-1)
	f.Close()
	if len(dirs) <= maxFileNum {
		return
	}
	sort.Slice(dirs, func(i, j int) bool {
		f1, _ := dirs[i].Info()
		f2, _ := dirs[j].Info()
		return f1.ModTime().Unix() > f2.ModTime().Unix()
	})
	index := strings.LastIndex(backupfileName, "_")
	indexSuffix := strings.LastIndex(backupfileName, ".")
	if indexSuffix == 0 {
		indexSuffix = len(backupfileName)
	}
	prefixname := backupfileName[:index+1]
	suffix := backupfileName[indexSuffix:]
	suffixlen := len(suffix)
	rmfiles := make([]string, 0)
	i := 0
	for _, ff := range dirs {
		checkfname := ff.Name()
		if isGzip && strings.HasSuffix(checkfname, ".gz") {
			checkfname = checkfname[:len(checkfname)-3]
		}
		if len(checkfname) > len(prefixname) && checkfname[:len(prefixname)] == prefixname && matchString("^[0-9]+$", checkfname[len(prefixname):len(checkfname)-suffixlen]) {
			finfo, err := ff.Info()
			if err == nil && !finfo.IsDir() {
				i++
				if i > maxFileNum {
					rmfiles = append(rmfiles, fmt.Sprint(dir, "/", f.Name()))
				}
			}
		}
	}
	if len(rmfiles) > 0 {
		for _, k := range rmfiles {
			os.Remove(k)
		}
	}
}
func (t *fileStore) getBackupDailyFileName(dir, filename string, mode TimeMode, isGzip bool) (bckupfilename string) {
	timeStr := _yestStr(mode)
	index := strings.LastIndex(filename, ".")
	if index <= 0 {
		index = len(filename)
	}
	fname := filename[:index]
	suffix := filename[index:]
	bckupfilename = fmt.Sprint(fname, "_", timeStr, suffix)
	if isGzip {
		if isFileExist(fmt.Sprint(t.fileDir, "/", bckupfilename, ".gz")) {
			bckupfilename = t.getBackupFilename(1, dir, fmt.Sprint(fname, "_", timeStr), suffix, isGzip)
		}
	} else {
		if isFileExist(fmt.Sprint(dir, "/", bckupfilename)) {
			bckupfilename = t.getBackupFilename(1, dir, fmt.Sprint(fname, "_", timeStr), suffix, isGzip)
		}
	}
	return
}
func (t *fileStore) getBackupFilename(count int, dir, filename, suffix string, isGzip bool) (backupFilename string) {
	backupFilename = fmt.Sprint(filename, "_", count, suffix)
	if isGzip {
		if isFileExist(fmt.Sprint(dir, "/", backupFilename, ".gz")) {
			return t.getBackupFilename(count+1, dir, filename, suffix, isGzip)
		}
	} else {
		if isFileExist(fmt.Sprint(dir, "/", backupFilename)) {
			return t.getBackupFilename(count+1, dir, filename, suffix, isGzip)
		}
	}
	return
}
func (t *fileStore) getBackupRollFileName(dir, filename string, isGzip bool) (backupFilename string, er error) {
	list, err := getDirList(dir)
	if err != nil {
		er = err
		return
	}
	index := strings.LastIndex(filename, ".")
	if index <= 0 {
		index = len(filename)
	}
	fname := filename[:index]
	suffix := filename[index:]
	i := 1
	for _, fd := range list {
		pattern := fmt.Sprint(`^`, fname, `_[\d]{1,}`, suffix, `$`)
		if isGzip {
			pattern = fmt.Sprint(`^`, fname, `_[\d]{1,}`, suffix, `.gz$`)
		}
		if matchString(pattern, fd.Name()) {
			i++
		}
	}
	backupFilename = t.getBackupFilename(i, dir, fname, suffix, isGzip)
	return
}
func (t *fileStore) close() (err error) {
	defer catchError()
	if t.fileHandler != nil {
		err = t.fileHandler.Close()
	}
	return
}

func tomorSecond(mode TimeMode) int64 {
	now := time.Now()
	switch mode {
	case ModeDay:
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()).Unix()
	case ModeHour:
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location()).Unix()
	case ModeMonth:
		return time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1).Unix()
	default:
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()).Unix()
	}
}
func _yestStr(mode TimeMode) string {
	now := time.Now()
	switch mode {
	case ModeDay:
		return now.AddDate(0, 0, -1).Format(dateformatDay)
	case ModeHour:
		return now.Add(-1 * time.Hour).Format(dateformatHour)
	case ModeMonth:
		return now.AddDate(0, -1, 0).Format(dateformatMonth)
	default:
		return now.AddDate(0, 0, -1).Format(dateformatDay)
	}
}
func lgzip(gzfile, gzname, srcfile string) (err error) {
	var gf *os.File
	if gf, err = os.Create(gzfile); err == nil {
		defer gf.Close()
		var f1 *os.File
		if f1, err = os.Open(srcfile); err == nil {
			defer f1.Close()
			gw := gzip.NewWriter(gf)
			defer gw.Close()
			gw.Header.Name = gzname
			var buf bytes.Buffer
			io.Copy(&buf, f1)
			_, err = gw.Write(buf.Bytes())
		}
	}
	return
}
func matchString(pattern string, s string) bool {
	b, err := regexp.MatchString(pattern, s)
	if err != nil {
		b = false
	}
	return b
}
func getDirList(dir string) ([]os.DirEntry, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadDir(-1)
}
