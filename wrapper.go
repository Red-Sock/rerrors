package errors

import (
	"fmt"
	"runtime"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Wrap(innerError error, msg ...any) error {
	str, grpcCode := split(msg)

	se, ok := status.FromError(innerError)
	if ok {
		if grpcCode == nil {
			c := se.Code()
			grpcCode = &c
		}
	}

	err := Error{
		innerError: innerError,
		msg:        strings.Join(str, "; "),
		grpcCode:   grpcCode,
	}

	if enableTracing {
		runtime.Callers(2, err.trace[:])
	}

	return err
}

func Wrapf(err error, msg string, args ...interface{}) error {
	return Wrap(err, fmt.Sprintf(msg, args...))
}

func split(in []any) (str []string, grpcCode *codes.Code) {
	for _, m := range in {
		switch v := m.(type) {
		case string:
			str = append(str, v)
		case codes.Code:
			grpcCode = &v
		}
	}

	return
}
