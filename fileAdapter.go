package logs

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type RollingType uint8

const (
	RollingDaily RollingType = 0 //按日期分割文件
	RollingFile  RollingType = 1 //按文件大小分割文件

	dateformatDay = "20060102"
)

const (
	_  = iota
	KB = 1 << (iota * 10)
	MB
	GB
	TB
)

type FileConfig struct {
	Dir         string
	FileName    string
	FileMaxSize int64
	FileMaxNum  int
	RollType    RollingType
	Gzip        bool
}

type FileLog struct {
	BaseAdapter
	cache   chan *Item
	fs      *fileStore
	sync    bool
	_rwLock sync.RWMutex
	cancel  chan struct{}
}

func NewFileLog(level LEVEL, cfg *FileConfig, async int, trimPath string) (Adapter, error) {
	f := &FileLog{
		BaseAdapter: BaseAdapter{
			level:      level,
			timeFormat: DefaultTimeFormatLong,
			format:     DefaultLogFormat,
			trimPath:   trimPath,
			trim:       len(trimPath) != 0,
			name:       "file",
		},
		fs: new(fileStore),
	}
	if async == 0 {
		f.sync = true
	} else {
		f.cache = make(chan *Item, async)
		f.sync = false
	}
	f.fs.fileDir = cfg.Dir
	f.fs.fileName = cfg.FileName
	f.fs.maxSize = cfg.FileMaxSize
	f.fs.rollType = cfg.RollType
	f.fs.maxFileNum = cfg.FileMaxNum
	f.fs.gzip = cfg.Gzip
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
	defer c.fs.close()
	buf := bytes.NewBuffer(nil)
	for {
		select {
		case item := <-c.cache:
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
		c.clean(item)
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
	defer c.clean(item)
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
	buf.WriteString("[")
	buf.WriteString(strconv.Itoa(item.Line))
	buf.WriteString("]:")
	buf.WriteString(item.Content)
	buf.WriteByte('\n')
	return buf
}

// 两种模式:
// 1.RollingDaily, 最大存储日志=TimeMode数量*maxFileNum.比如1天一个文件,最大存储日志=1天*maxFileNum
// 2.RollingFile, 最大存储日志=maxSize是单个文件的最大大小.最大存储日志=maxSize*maxFileNum
type fileStore struct {
	fileDir     string      //日志文件所在的目录
	fileName    string      //日志文件名
	maxSize     int64       //日志文件的最大大小
	fileSize    int64       //记录当前日志文件的大小
	rollType    RollingType //日志的滚动模式,按日期或按文件大小
	tomorSecond int64       //日志文件的下一天的时间戳
	isFileWell  bool        //文件是否正常
	maxFileNum  int         //日志文件最大数量
	gzip        bool        //是否开启gzip

	fileHandler *os.File
	lock        sync.Mutex
}

func (t *fileStore) openFileHandler() (e error) {
	if t.fileDir == "" || t.fileName == "" {
		e = errors.New("log filePath is null or error")
		return
	}
	e = os.MkdirAll(t.fileDir, 0666)
	if e != nil {
		t.isFileWell = false
		return
	}
	fname := filepath.Join(t.fileDir, t.fileName)
	t.fileHandler, e = os.OpenFile(fname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if e != nil {
		t.isFileWell = false
		return
	}
	t.isFileWell = true
	t.tomorSecond = tomorSecond()
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
		return t.maxSize > 0 && t.fileSize >= t.maxSize
	}
	return false
}
func (t *fileStore) rename() (backupFileName string, err error) {
	if t.rollType == RollingDaily {
		backupFileName = t.getBackupDailyFileName(t.fileDir, t.fileName, t.gzip)
	} else {
		backupFileName, err = t.getBackupRollFileName(t.fileDir, t.fileName, t.gzip)
	}
	if backupFileName != "" && err == nil {
		oldPath := filepath.Join(t.fileDir, t.fileName)
		newPath := filepath.Join(t.fileDir, backupFileName)
		err = os.Rename(oldPath, newPath)
		if err == nil {
			if t.gzip {
				go func() {
					if err = lgzip(fmt.Sprint(newPath, ".gz"), backupFileName, newPath); err == nil {
						os.Remove(newPath)
					}
					if err == nil && t.maxFileNum > 0 {
						t.rmDeadFile(t.fileDir, backupFileName, t.maxFileNum, t.gzip)
					}
				}()
			} else {
				t.rmDeadFile(t.fileDir, backupFileName, t.maxFileNum, t.gzip)
			}
		}
	}
	return
}
func (t *fileStore) rmDeadFile(dir, backupFileName string, maxFileNum int, isGzip bool) {
	index := strings.LastIndex(backupFileName, "_")
	indexSuffix := strings.LastIndex(backupFileName, ".")
	if indexSuffix == 0 {
		indexSuffix = len(backupFileName)
	}
	preFixname := backupFileName[:index+1]
	suffix := backupFileName[indexSuffix:]
	ret, err := getDir(dir, preFixname, suffix)
	if len(ret) <= maxFileNum {
		return
	}
	if err != nil {
		return
	}
	for i := maxFileNum; i < len(ret); i++ {
		os.Remove(filepath.Join(dir, ret[i].Name()))
		fmt.Println("remove file:", filepath.Join(dir, ret[i].Name()))
	}
}
func (t *fileStore) getBackupDailyFileName(dir, filename string, isGzip bool) (bckupfilename string) {
	timeStr := _yestStr()
	index := strings.LastIndex(filename, ".")
	if index <= 0 {
		index = len(filename)
	}
	fname := filename[:index]
	suffix := filename[index:]
	bckupfilename = fmt.Sprint(fname, "_", timeStr, suffix)
	if isGzip {
		if isFileExist(filepath.Join(t.fileDir, bckupfilename) + ".gz") {
			bckupfilename = t.getBackupFilename(1, dir, fmt.Sprint(fname, "_", timeStr), suffix, isGzip)
		}
	} else {
		if isFileExist(filepath.Join(dir, bckupfilename)) {
			bckupfilename = t.getBackupFilename(1, dir, fmt.Sprint(fname, "_", timeStr), suffix, isGzip)
		}
	}
	return
}
func (t *fileStore) getBackupFilename(count int, dir, filename, suffix string, isGzip bool) (backupFilename string) {
	backupFilename = fmt.Sprint(filename, "-", count, suffix)
	if isGzip {
		if isFileExist(filepath.Join(dir, backupFilename) + ".gz") {
			return t.getBackupFilename(count+1, dir, filename, suffix, isGzip)
		}
	} else {
		if isFileExist(filepath.Join(dir, backupFilename)) {
			return t.getBackupFilename(count+1, dir, filename, suffix, isGzip)
		}
	}
	return
}
func (t *fileStore) getBackupRollFileName(dir, filename string, isGzip bool) (backupFilename string, er error) {
	timeStr := _yestStr()
	index := strings.LastIndex(filename, ".")
	if index <= 0 {
		index = len(filename)
	}
	fname := filename[:index]
	suffix := filename[index:]
	backupFilename = t.getBackupFilename(1, dir, fmt.Sprint(fname, "_", timeStr), suffix, isGzip)
	return
}
func (t *fileStore) close() (err error) {
	defer catchError()
	if t.fileHandler != nil {
		err = t.fileHandler.Close()
	}
	return
}

func tomorSecond() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()).Unix()
}
func _yestStr() string {
	return time.Now().Format(dateformatDay)
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

func isFileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
func matchString(pattern string, s string) bool {
	b, err := regexp.MatchString(pattern, s)
	if err != nil {
		b = false
	}
	return b
}

// eg:prefix="mylog_",suffix=".log"
func getPattern(prefix, suffix string) string {
	var builder strings.Builder
	builder.WriteString(strings.Replace(strings.Replace(prefix, "_", `\_`, -1), "-", `\-`, -1))
	builder.WriteString(`\d+(\-\d+)?\`)
	if len(suffix) > 0 {
		builder.WriteString(suffix)
	}
	builder.WriteString(`(\.gz)?`)
	return builder.String()
}

func getDir(dir, prefix, suffix string) (ret []os.DirEntry, err error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	list, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	pattern := getPattern(prefix, suffix)
	for i := 0; i < len(list); i++ {
		if !list[i].IsDir() {
			if matchString(pattern, list[i].Name()) {
				ret = append(ret, list[i])
			}
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		f1, err := ret[i].Info()
		if err != nil {
			return false
		}
		f2, err := ret[j].Info()
		if err != nil {
			return false
		}
		return f1.ModTime().UnixMilli() > f2.ModTime().UnixMilli()
	})
	return ret, nil
}
