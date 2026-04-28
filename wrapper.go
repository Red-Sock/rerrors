package rerrors

import (
	"fmt"
	"runtime"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/protoadapt"
)

func Wrap(innerError error, msg ...any) error {
	ev := split(msg)

	err := Error{
		innerError: innerError,
		msg:        strings.Join(ev.str, "; "),
		grpcCode:   ev.grpcCode,
	}

	if enableTracing {
		runtime.Callers(2, err.trace[:])
	}

	return err
}

func Wrapf(err error, msg string, args ...interface{}) error {
	return Wrap(err, fmt.Sprintf(msg, args...))
}

type errorValues struct {
	str         []string
	grpcCode    *codes.Code
	grpcDetails []protoadapt.MessageV1

	opts []opt
}

func split(in []any) (ev errorValues) {
	for _, m := range in {
		switch v := m.(type) {
		case string:
			ev.str = append(ev.str, v)
		case codes.Code:
			ev.grpcCode = &v
		case opt:
			ev.opts = append(ev.opts, v)

		case protoadapt.MessageV1:
			ev.grpcDetails = append(ev.grpcDetails, v)
		}
	}

	return ev
}
