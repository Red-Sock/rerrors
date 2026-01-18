package rerrors

import (
	"errors"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
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
	ev := split(args)

	err := Error{
		msg:      strings.Join(append([]string{msg}, ev.str...), "; "),
		grpcCode: ev.grpcCode,
	}

	if enableTracing {
		runtime.Callers(2, err.trace[:])
	}

	return err
}

func NewUserError(msg string, args ...any) error {
	ev := split(args)

	err := Error{
		msg:         strings.Join(append([]string{msg}, ev.str...), "; "),
		grpcCode:    ev.grpcCode,
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
	httpCode *int
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
	errSeparator := GetSeparator()

	frames := runtime.CallersFrames(e.trace[:])
	fr, ok := frames.Next()
	if ok {
		traceStr := strings.Join(
			[]string{fr.Function + "() returned -> \"" + e.msg + "\"",
				"        " + fr.File + ":" + strconv.Itoa(fr.Line)}, "\n")
		msg = errSeparator + traceStr
	}

	if e.innerError != nil {
		var cE Error
		if errors.As(e.innerError, &cE) {
			msg = cE.errorWithTrace() + errSeparator + msg
		} else {
			msg = e.innerError.Error() + errSeparator + msg
		}
	}

	return msg
}

func (e Error) error() (msg string) {
	msg += e.msg
	errSeparator := GetSeparator()

	if e.innerError != nil {
		var cE Error
		if errors.As(e.innerError, &cE) {
			msg = cE.error() + errSeparator + msg
		} else {
			msg = e.innerError.Error() + errSeparator + msg
		}
	}

	return msg
}

func (e Error) Unwrap() error {
	return e.innerError
}

func (e Error) GRPCStatus() *status.Status {
	var ie Error
	ok := errors.As(e.innerError, &ie)
	if ok {
		innerStat := ie.GRPCStatus()
		return status.New(innerStat.Code(), e.Error())
	}

	if e.grpcCode != nil {
		return status.New(*e.grpcCode, e.Error())
	}

	return status.New(codes.Internal, e.UserError())
}

func (e Error) HttpStatus(w http.ResponseWriter) {
	code := http.StatusInternalServerError
	if e.httpCode != nil {
		code = *e.httpCode
	}

	w.WriteHeader(code)
	_, _ = w.Write([]byte(e.innerError.Error()))
}

func Is(err1, err2 error) bool {
	return errors.Is(err1, err2)
}

func As(err1 error, err2 any) bool {
	return errors.As(err1, err2)
}

func Join(errs ...error) error {
	return errors.Join(errs...)
}

func addTraceDebugToGrpcStatus(s *status.Status, e Error) *status.Status {
	det := &errdetails.DebugInfo{
		Detail: e.errorWithTrace(),
	}

	s, _ = s.WithDetails(det)
	return s
}
