package rerrors

import (
	"testing"

	"github.com/stretchr/testify/require"
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
