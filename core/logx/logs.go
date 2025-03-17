package logx

import (
    "encoding/json" // Added for JsonFormatter
    "fmt"
    "io"
    "log"
    "os"
    "path"
    "reflect"
    "runtime/debug"
    "sync"
    "sync/atomic"
    "time"

    "github.com/zeromicro/go-zero/core/sysx"
)

const callerDepth = 4

// Added Formatter interface and implementations
type Formatter interface {
    Format(entry logEntry) ([]byte, error)
}

type PlainFormatter struct{}
func (f *PlainFormatter) Format(entry logEntry) ([]byte, error) {
    var buf []byte
    if t, ok := entry["@timestamp"]; ok {
        buf = append(buf, fmt.Sprintf("%v ", t)...)
    }
    if lvl, ok := entry["level"]; ok {
        buf = append(buf, fmt.Sprintf("%v ", lvl)...)
    }
    if msg, ok := entry["message"]; ok {
        buf = append(buf, fmt.Sprintf("%v", msg)...)
    }
    for k, v := range entry {
        if k != "@timestamp" && k != "level" && k != "message" {
            buf = append(buf, fmt.Sprintf(" %s=%v", k, v)...)
        }
    }
    buf = append(buf, '\n')
    return buf, nil
}

type JsonFormatter struct{}
func (f *JsonFormatter) Format(entry logEntry) ([]byte, error) {
    return json.Marshal(entry)
}

var (
    timeFormat        = "2006-01-02T15:04:05.000Z07:00"
    // Removed: encoding uint32 = jsonEncodingType
    defaultFormatter Formatter = &JsonFormatter{} // Added: Default to JSON
    formatter        atomic.Value                 // Added: Atomic formatter storage
    maxContentLength uint32
    disableStat      uint32
    logLevel         uint32
    options          logOptions
    writer           = new(atomicWriter)
    setupOnce        sync.Once
)

type (
    LogField struct {
        Key   string
        Value any
    }
    LogOption func(options *logOptions)
    logEntry  map[string]any
    logOptions struct {
        gzipEnabled           bool
        logStackCooldownMills int
        keepDays              int
        maxBackups            int
        maxSize               int
        rotationRule          string
    }
)

func init() {
    formatter.Store(defaultFormatter) // Initialize formatter with default
}

// SetFormatter sets the log formatter dynamically.
func SetFormatter(f Formatter) {
    if f == nil {
        formatter.Store(defaultFormatter)
    } else {
        formatter.Store(f)
    }
}

// AddWriter remains unchanged
func AddWriter(w Writer) {
    ow := Reset()
    if ow == nil {
        SetWriter(w)
    } else {
        SetWriter(comboWriter{
            writers: []Writer{ow, w},
        })
    }
}

// Alert remains unchanged
func Alert(v string) {
    getWriter().Alert(v)
}

// Close remains unchanged
func Close() error {
    if w := writer.Swap(nil); w != nil {
        return w.(io.Closer).Close()
    }
    return nil
}

// Debug remains unchanged
func Debug(v ...any) {
    if shallLog(DebugLevel) {
        writeDebug(fmt.Sprint(v...))
    }
}

// Debugf remains unchanged
func Debugf(format string, v ...any) {
    if shallLog(DebugLevel) {
        writeDebug(fmt.Sprintf(format, v...))
    }
}

// Debugfn remains unchanged
func Debugfn(fn func() any) {
    if shallLog(DebugLevel) {
        writeDebug(fn())
    }
}

// Debugv remains unchanged
func Debugv(v any) {
    if shallLog(DebugLevel) {
        writeDebug(v)
    }
}

// Debugw remains unchanged
func Debugw(msg string, fields ...LogField) {
    if shallLog(DebugLevel) {
        writeDebug(msg, fields...)
    }
}

// Disable remains unchanged
func Disable() {
    atomic.StoreUint32(&logLevel, disableLevel)
    writer.Store(nopWriter{})
}

// DisableStat remains unchanged
func DisableStat() {
    atomic.StoreUint32(&disableStat, 1)
}

// Error remains unchanged
func Error(v ...any) {
    if shallLog(ErrorLevel) {
        writeError(fmt.Sprint(v...))
    }
}

// Errorf remains unchanged
func Errorf(format string, v ...any) {
    if shallLog(ErrorLevel) {
        writeError(fmt.Errorf(format, v...).Error())
    }
}

// Errorfn remains unchanged
func Errorfn(fn func() any) {
    if shallLog(ErrorLevel) {
        writeError(fn())
    }
}

// ErrorStack remains unchanged
func ErrorStack(v ...any) {
    if shallLog(ErrorLevel) {
        writeStack(fmt.Sprint(v...))
    }
}

// ErrorStackf remains unchanged
func ErrorStackf(format string, v ...any) {
    if shallLog(ErrorLevel) {
        writeStack(fmt.Sprintf(format, v...))
    }
}

// Errorv remains unchanged
func Errorv(v any) {
    if shallLog(ErrorLevel) {
        writeError(v)
    }
}

// Errorw remains unchanged
func Errorw(msg string, fields ...LogField) {
    if shallLog(ErrorLevel) {
        writeError(msg, fields...)
    }
}

// Field remains unchanged
func Field(key string, value any) LogField {
    switch val := value.(type) {
    case error:
        return LogField{Key: key, Value: encodeError(val)}
    case []error:
        var errs []string
        for _, err := range val {
            errs = append(errs, encodeError(err))
        }
        return LogField{Key: key, Value: errs}
    case time.Duration:
        return LogField{Key: key, Value: fmt.Sprint(val)}
    case []time.Duration:
        var durs []string
        for _, dur := range val {
            durs = append(durs, fmt.Sprint(dur))
        }
        return LogField{Key: key, Value: durs}
    case []time.Time:
        var times []string
        for _, t := range val {
            times = append(times, fmt.Sprint(t))
        }
        return LogField{Key: key, Value: times}
    case fmt.Stringer:
        return LogField{Key: key, Value: encodeStringer(val)}
    case []fmt.Stringer:
        var strs []string
        for _, str := range val {
            strs = append(strs, encodeStringer(str))
        }
        return LogField{Key: key, Value: strs}
    default:
        return LogField{Key: key, Value: val}
    }
}

// Info remains unchanged
func Info(v ...any) {
    if shallLog(InfoLevel) {
        writeInfo(fmt.Sprint(v...))
    }
}

// Infof remains unchanged
func Infof(format string, v ...any) {
    if shallLog(InfoLevel) {
        writeInfo(fmt.Sprintf(format, v...))
    }
}

// Infofn remains unchanged
func Infofn(fn func() any) {
    if shallLog(InfoLevel) {
        writeInfo(fn())
    }
}

// Infov remains unchanged
func Infov(v any) {
    if shallLog(InfoLevel) {
        writeInfo(v)
    }
}

// Infow remains unchanged
func Infow(msg string, fields ...LogField) {
    if shallLog(InfoLevel) {
        writeInfo(msg, fields...)
    }
}

// Must remains unchanged
func Must(err error) {
    if err == nil {
        return
    }
    msg := fmt.Sprintf("%+v\n\n%s", err.Error(), debug.Stack())
    log.Print(msg)
    getWriter().Severe(msg)
    if ExitOnFatal.True() {
        os.Exit(1)
    } else {
        panic(msg)
    }
}

// MustSetup remains unchanged
func MustSetup(c LogConf) {
    Must(SetUp(c))
}

// Reset remains unchanged
func Reset() Writer {
    return writer.Swap(nil)
}

// SetLevel remains unchanged
func SetLevel(level uint32) {
    atomic.StoreUint32(&logLevel, level)
}

// SetWriter remains unchanged
func SetWriter(w Writer) {
    if atomic.LoadUint32(&logLevel) != disableLevel {
        writer.Store(w)
    }
}

// SetUp updated to remove encoding references
func SetUp(c LogConf) (err error) {
    setupOnce.Do(func() {
        setupLogLevel(c)
        if !c.Stat {
            DisableStat()
        }
        if len(c.TimeFormat) > 0 {
            timeFormat = c.TimeFormat
        }
        if len(c.FileTimeFormat) > 0 {
            fileTimeFormat = c.FileTimeFormat
        }
        atomic.StoreUint32(&maxContentLength, c.MaxContentLength)
        // Removed: Encoding switch block
        switch c.Mode {
        case fileMode:
            err = setupWithFiles(c)
        case volumeMode:
            err = setupWithVolume(c)
        default:
            setupWithConsole()
        }
    })
    return
}

// Severe remains unchanged
func Severe(v ...any) {
    if shallLog(SevereLevel) {
        writeSevere(fmt.Sprint(v...))
    }
}

// Severef remains unchanged
func Severef(format string, v ...any) {
    if shallLog(SevereLevel) {
        writeSevere(fmt.Sprintf(format, v...))
    }
}

// Slow remains unchanged
func Slow(v ...any) {
    if shallLog(ErrorLevel) {
        writeSlow(fmt.Sprint(v...))
    }
}

// Slowf remains unchanged
func Slowf(format string, v ...any) {
    if shallLog(ErrorLevel) {
        writeSlow(fmt.Sprintf(format, v...))
    }
}

// Slowfn remains unchanged
func Slowfn(fn func() any) {
    if shallLog(ErrorLevel) {
        writeSlow(fn())
    }
}

// Slowv remains unchanged
func Slowv(v any) {
    if shallLog(ErrorLevel) {
        writeSlow(v)
    }
}

// Sloww remains unchanged
func Sloww(msg string, fields ...LogField) {
    if shallLog(ErrorLevel) {
        writeSlow(msg, fields...)
    }
}

// Stat remains unchanged
func Stat(v ...any) {
    if shallLogStat() && shallLog(InfoLevel) {
        writeStat(fmt.Sprint(v...))
    }
}

// Statf remains unchanged
func Statf(format string, v ...any) {
    if shallLogStat() && shallLog(InfoLevel) {
        writeStat(fmt.Sprintf(format, v...))
    }
}

// WithCooldownMillis remains unchanged
func WithCooldownMillis(millis int) LogOption {
    return func(opts *logOptions) {
        opts.logStackCooldownMills = millis
    }
}

// WithKeepDays remains unchanged
func WithKeepDays(days int) LogOption {
    return func(opts *logOptions) {
        opts.keepDays = days
    }
}

// WithGzip remains unchanged
func WithGzip() LogOption {
    return func(opts *logOptions) {
        opts.gzipEnabled = true
    }
}

// WithMaxBackups remains unchanged
func WithMaxBackups(count int) LogOption {
    return func(opts *logOptions) {
        opts.maxBackups = count
    }
}

// WithMaxSize remains unchanged
func WithMaxSize(size int) LogOption {
    return func(opts *logOptions) {
        opts.maxSize = size
    }
}

// WithRotation remains unchanged
func WithRotation(r string) LogOption {
    return func(opts *logOptions) {
        opts.rotationRule = r
    }
}

// Updated write functions to use Formatter
func writeDebug(val any, fields ...LogField) {
    getWriter().Debug(formatEntry("debug", val, fields...), mergeGlobalFields(addCaller(fields...))...)
}

func writeError(val any, fields ...LogField) {
    getWriter().Error(formatEntry("error", val, fields...), mergeGlobalFields(addCaller(fields...))...)
}

func writeInfo(val any, fields ...LogField) {
    getWriter().Info(formatEntry("info", val, fields...), mergeGlobalFields(addCaller(fields...))...)
}

func writeSevere(msg string) {
    getWriter().Severe(fmt.Sprintf("%s\n%s", msg, string(debug.Stack())))
}

func writeSlow(val any, fields ...LogField) {
    getWriter().Slow(formatEntry("slow", val, fields...), mergeGlobalFields(addCaller(fields...))...)
}

func writeStack(msg string) {
    getWriter().Stack(fmt.Sprintf("%s\n%s", msg, string(debug.Stack())))
}

func writeStat(msg string) {
    getWriter().Stat(msg, mergeGlobalFields(addCaller())...)
}

// Added formatEntry to use Formatter
func formatEntry(level string, val any, fields ...LogField) []byte {
    entry := make(logEntry)
    entry["@timestamp"] = time.Now().Format(timeFormat)
    entry["level"] = level
    switch v := val.(type) {
    case string:
        entry["message"] = v
    default:
        entry["message"] = fmt.Sprint(val)
    }
    for _, f := range fields {
        entry[f.Key] = f.Value
    }
    f := formatter.Load().(Formatter)
    data, err := f.Format(entry)
    if err != nil {
        log.Printf("Failed to format log entry: %v", err)
        return []byte(fmt.Sprintf("%v", val)) // Fallback
    }
    return data
}

// Remaining functions unchanged
func addCaller(fields ...LogField) []LogField {
    return append(fields, Field(callerKey, getCaller(callerDepth)))
}

func createOutput(path string) (io.WriteCloser, error) {
    if len(path) == 0 {
        return nil, ErrLogPathNotSet
    }
    var rule RotateRule
    switch options.rotationRule {
    case sizeRotationRule:
        rule = NewSizeLimitRotateRule(path, backupFileDelimiter, options.keepDays, options.maxSize,
            options.maxBackups, options.gzipEnabled)
    default:
        rule = DefaultRotateRule(path, backupFileDelimiter, options.keepDays, options.gzipEnabled)
    }
    return NewLogger(path, rule, options.gzipEnabled)
}

func encodeError(err error) (ret string) {
    return encodeWithRecover(err, func() string {
        return err.Error()
    })
}

func encodeStringer(v fmt.Stringer) (ret string) {
    return encodeWithRecover(v, func() string {
        return v.String()
    })
}

func encodeWithRecover(arg any, fn func() string) (ret string) {
    defer func() {
        if err := recover(); err != nil {
            if v := reflect.ValueOf(arg); v.Kind() == reflect.Ptr && v.IsNil() {
                ret = nilAngleString
            } else {
                ret = fmt.Sprintf("panic: %v", err)
            }
        }
    }()
    return fn()
}

func getWriter() Writer {
    w := writer.Load()
    if w == nil {
        w = writer.StoreIfNil(newConsoleWriter())
    }
    return w
}

func handleOptions(opts []LogOption) {
    for _, opt := range opts {
        opt(&options)
    }
}

func setupLogLevel(c LogConf) {
    switch c.Level {
    case levelDebug:
        SetLevel(DebugLevel)
    case levelInfo:
        SetLevel(InfoLevel)
    case levelError:
        SetLevel(ErrorLevel)
    case levelSevere:
        SetLevel(SevereLevel)
    }
}

func setupWithConsole() {
    SetWriter(newConsoleWriter())
}

func setupWithFiles(c LogConf) error {
    w, err := newFileWriter(c)
    if err != nil {
        return err
    }
    SetWriter(w)
    return nil
}

func setupWithVolume(c LogConf) error {
    if len(c.ServiceName) == 0 {
        return ErrLogServiceNameNotSet
    }
    c.Path = path.Join(c.Path, c.ServiceName, sysx.Hostname())
    return setupWithFiles(c)
}

func shallLog(level uint32) bool {
    return atomic.LoadUint32(&logLevel) <= level
}

func shallLogStat() bool {
    return atomic.LoadUint32(&disableStat) == 0
}