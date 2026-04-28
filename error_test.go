package rerrors

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
)

func Test_SimpleErrors(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		expected := "simple one message error"
		e := New(expected)

		actual := e.Error()

		require.Equal(t, expected, actual)
	})
	t.Run("simple_is", func(t *testing.T) {
		expected := "simple one message error"
		e := New(expected)

		require.ErrorIs(t, e, e)

		eWrapped := Wrap(e, "wrapped")
		require.ErrorIs(t, eWrapped, e)
	})
	t.Run("simple_empty_wrapped", func(t *testing.T) {
		expected := "simple one message error"
		e := New(expected)
		e = Wrap(e)
		actual := e.Error()

		require.Equal(t, expected, actual)
	})
	t.Run("simple_wrapped_with_message", func(t *testing.T) {
		rootMessage := "simple one message error"
		wrapper := "message wrapper"
		e := New(rootMessage)
		e = Wrap(e, wrapper)
		actual := e.Error()

		expected := rootMessage + ";" + wrapper

		require.Equal(t, expected, actual)
	})
	t.Run("grpc_code_in_error_message", func(t *testing.T) {
		msg := "resource not found"
		e := New(msg, codes.NotFound)
		errStr := e.Error()
		require.Contains(t, errStr, msg)
		require.Contains(t, errStr, "GrpcCode: "+strconv.Itoa(int(codes.NotFound)))
	})
	t.Run("grpc_with_details", func(t *testing.T) {
		rootMessage := "grpc request went wrong"

		e := New(rootMessage, codes.InvalidArgument, &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "name",
					Description: "contains invalid characters",
				},
			},
		})
		errStr := e.Error()
		require.Contains(t, errStr, rootMessage)
		require.Contains(t, errStr, "GrpcCode: "+strconv.Itoa(int(codes.InvalidArgument)))
		require.Contains(t, errStr, "GrpcDetails[0]:")
		require.Contains(t, errStr, "name")
	})
	t.Run("grpc_multiple_details_indexed", func(t *testing.T) {
		e := New("validation failed", codes.InvalidArgument,
			&errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "name", Description: "too short"},
			}},
			&errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "email", Description: "invalid format"},
			}},
		)
		errStr := e.Error()
		require.Contains(t, errStr, "GrpcDetails[0]:")
		require.Contains(t, errStr, "GrpcDetails[1]:")
	})
}

func Test_UserError(t *testing.T) {
	t.Parallel()

	t.Run("user_error_returns_own_message", func(t *testing.T) {
		msg := "invalid input provided"
		e := NewUserError(msg).(Error)
		require.Equal(t, msg, e.UserError())
	})
	t.Run("non_user_error_returns_full_error", func(t *testing.T) {
		msg := "internal failure"
		e := New(msg).(Error)
		require.Equal(t, msg, e.UserError())
	})
	t.Run("inner_user_error_propagates_through_wrap", func(t *testing.T) {
		userMsg := "user facing message"
		inner := NewUserError(userMsg)
		outer := Wrap(inner, "internal details")
		require.Equal(t, userMsg, outer.(Error).UserError())
	})
	t.Run("wrap_over_non_user_error_returns_full_message", func(t *testing.T) {
		inner := New("root cause")
		outer := Wrap(inner, "outer context")
		errStr := outer.(Error).UserError()
		require.Contains(t, errStr, "root cause")
	})
}

func Test_Wrapf(t *testing.T) {
	t.Parallel()

	t.Run("formats_message_with_args", func(t *testing.T) {
		inner := New("base error")
		wrapped := Wrapf(inner, "context: %s %d", "value", 42)
		errStr := wrapped.Error()
		require.Contains(t, errStr, "base error")
		require.Contains(t, errStr, "context: value 42")
	})
}

func Test_HttpStatus(t *testing.T) {
	t.Parallel()

	t.Run("default_500_when_no_code", func(t *testing.T) {
		e := New("internal error").(Error)
		w := httptest.NewRecorder()
		e.HttpStatus(w)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("custom_http_code_is_written", func(t *testing.T) {
		e := New("not found", WithHttpStatus(http.StatusNotFound)).(Error)
		w := httptest.NewRecorder()
		e.HttpStatus(w)
		require.Equal(t, http.StatusNotFound, w.Code)
	})
	t.Run("inner_http_code_propagates_through_wrap", func(t *testing.T) {
		inner := New("not found", WithHttpStatus(http.StatusNotFound))
		outer := Wrap(inner, "outer context")
		w := httptest.NewRecorder()
		outer.(Error).HttpStatus(w)
		require.Equal(t, http.StatusNotFound, w.Code)
	})
	t.Run("body_contains_error_message", func(t *testing.T) {
		e := New("resource not found", WithHttpStatus(http.StatusNotFound)).(Error)
		w := httptest.NewRecorder()
		e.HttpStatus(w)
		require.Contains(t, w.Body.String(), "resource not found")
	})
}
