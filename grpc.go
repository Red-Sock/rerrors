package errors

import (
	"google.golang.org/grpc/codes"
)

func WithGrpcStatus(st codes.Code, err error) error {
	e, ok := err.(Error)
	if !ok {
		e = Error{
			innerError: err,
		}
	}

	e.grpcCode = &st
	return e
}
