package rerrors

import (
	"errors"
	"testing"
	"time"

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

func Test_GrpcDetailOpts(t *testing.T) {
	t.Parallel()

	details := func(t *testing.T, args ...any) []any {
		t.Helper()
		combined := append([]any{codes.InvalidArgument}, args...)
		e := New("err", combined...).(Error)
		return e.GRPCStatus().Details()
	}

	t.Run("with_bad_request", func(t *testing.T) {
		d := details(t, WithBadRequest("name", "too short"))
		require.Len(t, d, 1)
		br, ok := d[0].(*errdetails.BadRequest)
		require.True(t, ok)
		require.Len(t, br.FieldViolations, 1)
		require.Equal(t, "name", br.FieldViolations[0].Field)
		require.Equal(t, "too short", br.FieldViolations[0].Description)
	})
	t.Run("with_bad_request_multiple_calls", func(t *testing.T) {
		d := details(t,
			WithBadRequest("name", "too short"),
			WithBadRequest("email", "invalid format"),
		)
		require.Len(t, d, 2)
		br0, ok := d[0].(*errdetails.BadRequest)
		require.True(t, ok)
		require.Equal(t, "name", br0.FieldViolations[0].Field)
		br1, ok := d[1].(*errdetails.BadRequest)
		require.True(t, ok)
		require.Equal(t, "email", br1.FieldViolations[0].Field)
	})
	t.Run("with_error_info", func(t *testing.T) {
		d := details(t, WithErrorInfo("QUOTA_EXCEEDED", "example.com", map[string]string{"key": "val"}))
		require.Len(t, d, 1)
		ei, ok := d[0].(*errdetails.ErrorInfo)
		require.True(t, ok)
		require.Equal(t, "QUOTA_EXCEEDED", ei.Reason)
		require.Equal(t, "example.com", ei.Domain)
		require.Equal(t, "val", ei.Metadata["key"])
	})
	t.Run("with_retry_info", func(t *testing.T) {
		d := details(t, WithRetryInfo(5*time.Second))
		require.Len(t, d, 1)
		ri, ok := d[0].(*errdetails.RetryInfo)
		require.True(t, ok)
		require.Equal(t, int64(5), ri.RetryDelay.Seconds)
	})
	t.Run("with_debug_info", func(t *testing.T) {
		d := details(t, WithDebugInfo("something broke", "frame1", "frame2"))
		require.Len(t, d, 1)
		di, ok := d[0].(*errdetails.DebugInfo)
		require.True(t, ok)
		require.Equal(t, "something broke", di.Detail)
		require.Equal(t, []string{"frame1", "frame2"}, di.StackEntries)
	})
	t.Run("with_quota_failure", func(t *testing.T) {
		d := details(t, WithQuotaFailure("projects/123/quotas/READ", "limit exceeded"))
		require.Len(t, d, 1)
		qf, ok := d[0].(*errdetails.QuotaFailure)
		require.True(t, ok)
		require.Equal(t, "projects/123/quotas/READ", qf.Violations[0].Subject)
		require.Equal(t, "limit exceeded", qf.Violations[0].Description)
	})
	t.Run("with_precondition_failure", func(t *testing.T) {
		d := details(t, WithPreconditionFailure("TOS", "projects/123", "terms not accepted"))
		require.Len(t, d, 1)
		pf, ok := d[0].(*errdetails.PreconditionFailure)
		require.True(t, ok)
		require.Equal(t, "TOS", pf.Violations[0].Type)
		require.Equal(t, "projects/123", pf.Violations[0].Subject)
		require.Equal(t, "terms not accepted", pf.Violations[0].Description)
	})
	t.Run("with_request_info", func(t *testing.T) {
		d := details(t, WithRequestInfo("req-abc", "shard=1"))
		require.Len(t, d, 1)
		ri, ok := d[0].(*errdetails.RequestInfo)
		require.True(t, ok)
		require.Equal(t, "req-abc", ri.RequestId)
		require.Equal(t, "shard=1", ri.ServingData)
	})
	t.Run("with_resource_info", func(t *testing.T) {
		d := details(t, WithResourceInfo("Book", "books/42", "user@example.com", "not found"))
		require.Len(t, d, 1)
		ri, ok := d[0].(*errdetails.ResourceInfo)
		require.True(t, ok)
		require.Equal(t, "Book", ri.ResourceType)
		require.Equal(t, "books/42", ri.ResourceName)
		require.Equal(t, "user@example.com", ri.Owner)
		require.Equal(t, "not found", ri.Description)
	})
	t.Run("with_help", func(t *testing.T) {
		d := details(t, WithHelp("https://example.com/docs", "API docs"))
		require.Len(t, d, 1)
		h, ok := d[0].(*errdetails.Help)
		require.True(t, ok)
		require.Equal(t, "https://example.com/docs", h.Links[0].Url)
		require.Equal(t, "API docs", h.Links[0].Description)
	})
	t.Run("with_localized_message", func(t *testing.T) {
		d := details(t, WithLocalizedMessage("en-US", "Something went wrong"))
		require.Len(t, d, 1)
		lm, ok := d[0].(*errdetails.LocalizedMessage)
		require.True(t, ok)
		require.Equal(t, "en-US", lm.Locale)
		require.Equal(t, "Something went wrong", lm.Message)
	})
	t.Run("opts_mix_with_raw_proto_args", func(t *testing.T) {
		// raw proto args are collected by split() before opts are applied,
		// so ErrorInfo (raw) lands at index 0, BadRequest (opt) at index 1.
		d := details(t,
			WithBadRequest("name", "too short"),
			&errdetails.ErrorInfo{Reason: "VALIDATION", Domain: "example.com"},
		)
		require.Len(t, d, 2)
		_, ok1 := d[0].(*errdetails.ErrorInfo)
		require.True(t, ok1)
		_, ok2 := d[1].(*errdetails.BadRequest)
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
