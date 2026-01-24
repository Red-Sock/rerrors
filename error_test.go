package rerrors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_SimpleErrors(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		expected := "simple one message error"
		e := New(expected)

		actual := e.Error()

		require.Equal(t, expected, actual)
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
}
