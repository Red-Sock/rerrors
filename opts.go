package rerrors

type opt func(err *Error)

func WithHttpStatus(code int) opt {
	return func(e *Error) {
		e.httpCode = &code
	}
}
