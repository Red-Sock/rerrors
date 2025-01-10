package rerrors

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var enableTracing = false

var enableTracingFlag = "--enable-rscli-tracing"

func init() {
	// in case when project was compiled with
	// "rscliErrorTracingDisabled" build flag,
	// but we need traces
	for _, item := range os.Args {
		if item == enableTracingFlag {
			enableTracing = true
			return
		}
	}
}

func New(msg string, args ...any) error {
	str, grpcCode := split(args)

	err := Error{
		msg:      strings.Join(append([]string{msg}, str...), "; "),
		grpcCode: grpcCode,
	}

	if enableTracing {
		runtime.Callers(2, err.trace[:])
	}

	return err
}

func NewUserError(msg string, args ...any) error {
	str, grpcCode := split(args)

	err := Error{
		msg:         strings.Join(append([]string{msg}, str...), "; "),
		grpcCode:    grpcCode,
		isUserError: true,
	}

	if enableTracing {
		runtime.Callers(2, err.trace[:])
	}

	return err
}

type Error struct {
	innerError error

	isUserError bool
	msg         string
	trace       [3]uintptr

	grpcCode *codes.Code
}

func (e Error) Error() (msg string) {
	if enableTracing {
		return e.errorWithTrace()
	}

	return e.error()
}

func (e Error) UserError() string {
	if e.isUserError {
		return e.msg
	}
	var cE Error
	if errors.As(e.innerError, &cE) {
		return cE.UserError()
	}

	return e.error()
}

func (e Error) errorWithTrace() (msg string) {
	frames := runtime.CallersFrames(e.trace[:])
	fr, ok := frames.Next()
	if ok {
		traceStr := strings.Join(
			[]string{fr.Function + "() returned -> \"" + e.msg + "\"",
				"        " + fr.File + ":" + strconv.Itoa(fr.Line)}, "\n")
		msg = "\n" + traceStr
	}

	if e.innerError != nil {
		var cE Error
		if errors.As(e.innerError, &cE) {
			msg = cE.errorWithTrace() + "\n" + msg
		} else {
			msg = e.innerError.Error() + "\n" + msg
		}
	}

	return msg
}

func (e Error) error() (msg string) {
	msg += e.msg

	if e.innerError != nil {
		var cE Error
		if errors.As(e.innerError, &cE) {
			msg = cE.error() + "\n" + msg
		} else {
			msg = e.innerError.Error() + "\n" + msg
		}
	}

	return msg
}

func (e Error) Unwrap() error {
	return e.innerError
}

func (e Error) GRPCStatus() *status.Status {
	if e.grpcCode != nil {
		return status.New(*e.grpcCode, e.Error())
	}

	var ie Error
	ok := errors.As(e.innerError, &ie)
	if ok {
		return ie.GRPCStatus()
	}

	return status.New(codes.Internal, e.UserError())
}

func Is(err1, err2 error) bool {
	return errors.Is(err1, err2)
}

func As(err1, err2 error) bool {
	return errors.As(err1, err2)
}

func Join(errs ...error) error {
	return errors.Join(errs...)
}
