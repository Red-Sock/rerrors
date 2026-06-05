package rerrors

import (
	"errors"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func (e Error) GRPCStatus() *status.Status {
	var ie Error
	ok := errors.As(e.innerError, &ie)
	if ok {
		return ie.GRPCStatus()
	}

	if e.grpcCode != nil {
		st := status.New(*e.grpcCode, e.msg)
		details := e.collectGrpcDetails()

		detailedSt, _ := st.WithDetails(details...)
		if detailedSt != nil {
			st = detailedSt
		}

		return st
	}

	return status.New(codes.Internal, e.UserError())
}

func (e Error) HttpStatus(w http.ResponseWriter) {
	if e.innerError != nil {
		var cE Error
		if errors.As(e.innerError, &cE) {
			cE.HttpStatus(w)
			return
		}
	}

	code := http.StatusInternalServerError
	if e.httpCode != nil {
		code = *e.httpCode
	}

	w.WriteHeader(code)
	_, _ = w.Write([]byte(e.Error()))
}

func (e Error) Unwrap() error {
	return e.innerError
}
