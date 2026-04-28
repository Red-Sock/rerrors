package rerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_WithGrpc(t *testing.T) {
	t.Parallel()

	t.Run("unwrap_by_grpc", func(t *testing.T) {
		originalMessage := "order already created"

		originalErr := New(originalMessage, codes.AlreadyExists)

		wrappedMessage := "wrapped message"
		wrappedErr := Wrap(originalErr, wrappedMessage)

		statusErr, ok := status.FromError(wrappedErr)
		require.True(t, ok)

		gotMessage := statusErr.String()

		expectedMessage := "rpc error: code = AlreadyExists desc = order already created;wrapped message"
		require.Contains(t, gotMessage, originalMessage)
		require.Contains(t, gotMessage, wrappedMessage)
		require.Equal(t, expectedMessage, gotMessage)
	})
}

func Test_GRPCStatus(t *testing.T) {
	t.Parallel()

	t.Run("no_grpc_code_returns_internal", func(t *testing.T) {
		e := New("something went wrong").(Error)
		st := e.GRPCStatus()
		require.Equal(t, codes.Internal, st.Code())
	})
	t.Run("grpc_code_is_preserved", func(t *testing.T) {
		e := New("resource not found", codes.NotFound).(Error)
		st := e.GRPCStatus()
		require.Equal(t, codes.NotFound, st.Code())
	})
	t.Run("wrapped_error_inherits_inner_grpc_code", func(t *testing.T) {
		inner := New("original error", codes.PermissionDenied)
		outer := Wrap(inner, "additional context")
		st, ok := status.FromError(outer)
		require.True(t, ok)
		require.Equal(t, codes.PermissionDenied, st.Code())
	})
	t.Run("grpc_status_with_details", func(t *testing.T) {
		violation := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "email", Description: "invalid format"},
			},
		}
		e := New("bad request", codes.InvalidArgument, violation).(Error)
		st := e.GRPCStatus()

		require.Equal(t, codes.InvalidArgument, st.Code())

		details := st.Details()
		require.Len(t, details, 1)

		badReq, ok := details[0].(*errdetails.BadRequest)
		require.True(t, ok)
		require.Len(t, badReq.FieldViolations, 1)
		require.Equal(t, "email", badReq.FieldViolations[0].Field)
		require.Equal(t, "invalid format", badReq.FieldViolations[0].Description)
	})
	t.Run("grpc_status_multiple_details_preserved", func(t *testing.T) {
		e := New("error", codes.InvalidArgument,
			&errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "name", Description: "too short"},
			}},
			&errdetails.ErrorInfo{Reason: "QUOTA_EXCEEDED", Domain: "example.com"},
		).(Error)
		st := e.GRPCStatus()

		details := st.Details()
		require.Len(t, details, 2)

		_, ok1 := details[0].(*errdetails.BadRequest)
		require.True(t, ok1)
		_, ok2 := details[1].(*errdetails.ErrorInfo)
		require.True(t, ok2)
	})
}

func Test_WithGrpcStatus(t *testing.T) {
	t.Parallel()

	t.Run("plain_error_gets_grpc_code", func(t *testing.T) {
		plain := errors.New("plain error")
		wrapped := WithGrpcStatus(codes.NotFound, plain).(Error)
		st := wrapped.GRPCStatus()
		require.Equal(t, codes.NotFound, st.Code())
		require.Contains(t, st.Message(), "plain error")
	})
	t.Run("existing_rerror_code_is_replaced", func(t *testing.T) {
		e := New("existing error", codes.InvalidArgument)
		updated := WithGrpcStatus(codes.NotFound, e).(Error)
		st := updated.GRPCStatus()
		require.Equal(t, codes.NotFound, st.Code())
	})
}
