package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var goRoot = runtime.GOROOT()

var formatPartHead = []byte{'\n', '\t', '['}

const (
	formatPartColon = ':'
	formatPartTail  = ']'
	formatPartSpace = ' '
)

type Err struct {
	message  string
	stdError error
	// prevErr 指向上一个Err
	prevErr     *Err
	stack       []uintptr
	once        sync.Once
	fullMessage string
}

type stackFrame struct {
	funcName string
	file     string
	line     int
	message  string
}

// Error
func (e *Err) Error() string {
	e.once.Do(func() {
		var buf strings.Builder
		buf.Grow(512)
		var (
			messages []string
			stack    []uintptr
		)
		for prev := e; prev != nil; prev = prev.prevErr {
			stack = prev.stack
			if prev.stdError != nil {
				messages = append(messages, fmt.Sprintf("%s err:%s", prev.message, prev.stdError.Error()))
			} else {
				messages = append(messages, prev.message)
			}
		}
		sf := stackFrame{}
		for i, v := range stack {
			if j := len(messages) - 1 - i; j > -1 {
				sf.message = messages[j]
			} else {
				sf.message = ""
			}
			funcForPc := runtime.FuncForPC(v)
			if funcForPc == nil {
				sf.file = "???"
				sf.line = 0
				sf.funcName = "???"
				//fmt.Fprintf(buf, "\n\t[%s:%d:%s:%s]", sf.file, sf.line, sf.funcName, sf.message)
				buf.Write(formatPartHead)
				buf.WriteByte(formatPartSpace)
				buf.WriteString(sf.file)
				buf.WriteByte(formatPartColon)
				buf.WriteString(strconv.Itoa(sf.line))
				buf.WriteByte(formatPartSpace)
				buf.WriteString(sf.funcName)
				buf.WriteByte(formatPartColon)
				buf.WriteString(sf.message)
				buf.WriteByte(formatPartTail)
				continue
			}
			sf.file, sf.line = funcForPc.FileLine(v - 1)
			if strings.HasPrefix(sf.file, goRoot) {
				continue
			}

			// 处理函数名
			sf.funcName = funcForPc.Name()

			// 保证闭包函数名也能正确显示 如TestErrorf.func1:
			idx := strings.LastIndexByte(sf.funcName, '/')
			if idx != -1 {
				sf.funcName = sf.funcName[idx:]
				idx = strings.IndexByte(sf.funcName, '.')
				if idx != -1 {
					sf.funcName = strings.TrimPrefix(sf.funcName[idx:], ".")
				}
			}
			//fmt.Fprintf(buf, "\n\t[%s:%d:%s:%s]", sf.file, sf.line, sf.funcName, sf.message)
			buf.Write(formatPartHead)
			// 处理文件名行号时增加空格, 以便让IDE识别到, 可以点击跳转到源码.
			buf.WriteByte(formatPartSpace)
			buf.WriteString(sf.file)
			buf.WriteByte(formatPartColon)
			buf.WriteString(strconv.Itoa(sf.line))
			buf.WriteByte(formatPartSpace)
			buf.WriteString(sf.funcName)
			buf.WriteByte(formatPartColon)
			buf.WriteString(sf.message)
			buf.WriteByte(formatPartTail)
		}
		e.fullMessage = buf.String()
	})
	return e.fullMessage
}

// Prev 返回上一步传入Errorf的*Err
func (e *Err) Prev() *Err {
	return e.prevErr
}

// Inner 返回上一步传入Errorf的error, 用于判断error的值和已定义的error类型是否相等
func (e *Err) Inner() error {
	return e.stdError
}

// As
// 适配 errors.As errors.Is errors.Unwrap
func (e *Err) As(target interface{}) bool {
	if e.stdError != nil {
		return errors.As(e.stdError, target)
	}
	if e.prevErr != nil {
		return errors.As(e.prevErr, target)
	}
	// 最里层
	return errors.As(errors.New(e.message), target)
}

// Is
// 适配 errors.As errors.Is errors.Unwrap
func (e *Err) Is(target error) bool {
	if e.stdError != nil {
		return errors.Is(e.stdError, target)
	}
	if e.prevErr != nil {
		return errors.Is(e.prevErr, target)
	}
	// 最里层
	return errors.Is(errors.New(e.message), target)
}

// Unwrap
// 适配 errors.As errors.Is errors.Unwrap
func (e *Err) Unwrap() error {
	if e.stdError != nil {
		return errors.Unwrap(e.stdError)
	}
	if e.prevErr != nil {
		return errors.Unwrap(e.prevErr)
	}
	// 最里层
	return errors.New(e.message)
}

// New 是标准库中的New函数 只能用于定义error常量使用
func New(msg string) error {
	return errors.New(msg)
}

// As 是标准库中的As函数, 如果err是此包的*errors.Err类型, 那么实际上是判断最里层的err能不能转为target
// 相当于 if target,ok:=err.(targetType);ok { //如果err是wrap了的, 则继续判断unwrap的err直到不能unwrap为止 }
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Is 是标准库中的Is函数, 如果err是此包的*errors.Err类型, 那么实际上是判断最里层的err是不是target类型
// 相当于 if _,ok:=err.(targetType);ok { //如果err是wrap了的, 则继续判断unwrap的err直到不能unwrap为止 }
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// Unwrap 是标准库中的Unwrap函数
// 使用前必须能搞清楚它是怎么工作的, 否则不推荐使用
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Errorf
// 如果参数error类型不为*Err(error常量或自定义error类型或nil), 用于最早出错的地方, 会收集调用栈
// 如果参数error类型为*Err, 不会收集调用栈.
func Errorf(err error, format string, a ...interface{}) error {
	var msg string
	if len(a) == 0 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, a...)
	}
	if err, ok := err.(*Err); ok {
		return &Err{
			message: msg,
			prevErr: err,
		}
	}
	newErr := new_(msg)
	newErr.stdError = err
	return newErr
}

func new_(msg string) *Err {
	pc := make([]uintptr, 200)
	length := runtime.Callers(3, pc)
	return &Err{
		message: msg,
		stack:   pc[:length],
	}
}
