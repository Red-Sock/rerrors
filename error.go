package rerrors

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
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
	return newErr(msg, args...)
}

func NewUserError(msg string, args ...any) error {
	e := newErr(msg, args...)

	e.isUserError = true

	return e
}

func newErr(msg string, args ...any) Error {
	ev := split(args)

	err := Error{
		msg:         strings.Join(append([]string{msg}, ev.str...), "; "),
		grpcCode:    ev.grpcCode,
		grpcDetails: ev.grpcDetails,
	}

	for _, o := range ev.opts {
		o(&err)
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

	grpcCode    *codes.Code
	grpcDetails []protoadapt.MessageV1
	httpCode    *int
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
	msg = e.msg

	errSeparator := GetSeparator()

	if e.innerError != nil {
		var cE Error
		if errors.As(e.innerError, &cE) {
			msg = cE.error()
		} else {
			msg = e.innerError.Error()
		}

		if e.msg != "" {
			msg += errSeparator + e.msg
		}
	}

	if e.grpcCode != nil {
		msg += errSeparator + "GrpcCode: " + strconv.Itoa(int(*e.grpcCode))
	}

	if len(e.grpcDetails) != 0 {
		for idx, detail := range e.grpcDetails {
			msg += errSeparator + fmt.Sprintf("GrpcDetails[%d]:", idx) + detail.String()
		}
	}

	if e.httpCode != nil {
		msg += errSeparator + "HttpCode: " + strconv.Itoa(int(*e.httpCode))
	}

	return msg
}

func (e Error) collectGrpcDetails() []protoadapt.MessageV1 {
	var innerDetails []protoadapt.MessageV1

	if e.innerError != nil {
		var rerr Error
		if errors.As(e.innerError, &rerr) {
			innerDetails = rerr.collectGrpcDetails()
		}
	}

	return append(e.grpcDetails, innerDetails...)
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
