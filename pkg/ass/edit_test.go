package ass

import (
	"strings"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfoEditsRejectDuplicateCase(t *testing.T) {
	t.Parallel()

	document, err := Parse(strings.NewReader(fixture(t)))
	require.NoError(t, err)

	_, err = document.Apply(Edit{Info: map[string]string{"Title": "One", "title": "Two"}})
	require.ErrorIs(t, err, commerr.ErrInvalidValue)
}

func TestEditScriptStyleAndEvent(t *testing.T) {
	t.Parallel()

	document, err := Parse(strings.NewReader(fixture(t)))
	require.NoError(t, err)

	t.Run("applies every kind of edit", func(t *testing.T) {
		t.Parallel()

		updated, err := document.Apply(Edit{
			Info:   map[string]string{"Title": "New title", "WrapStyle": "1"},
			Styles: []StyleEdit{{Name: "Default", Fields: map[string]string{"FontSize": "44"}}},
			Events: []EventEdit{{
				Index: 1, Fields: map[string]string{"Start": "0:00:00.50", "Text": "Hi, world"},
				Override: &Override{Position: &Position{X: 12, Y: 34}},
			}},
		})
		require.NoError(t, err)

		result := updated.Contents()
		assert.Equal(t, "New title", result.Info["Title"])
		assert.Equal(t, "1", result.Info["WrapStyle"])
		require.Len(t, result.Styles, 1)
		assert.Equal(t, "44", result.Styles[0]["Fontsize"])
		require.Len(t, result.Events, 1)
		assert.Equal(t, "0:00:00.50", result.Events[0]["Start"])
		assert.Equal(t, "{\\pos(12,34)}Hi, world", result.Events[0]["Text"])
	})

	t.Run("rejects line breaks in a field", func(t *testing.T) {
		t.Parallel()

		_, err := document.Apply(Edit{
			Events: []EventEdit{{Index: 1, Fields: map[string]string{"Style": "Injected\r\nDialogue:"}}},
		})
		require.ErrorIs(t, err, commerr.ErrInvalidValue)
	})
}
