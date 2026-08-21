package site_model_calls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNormalizesModels(t *testing.T) {
	config, err := Parse(`{"enabled":true,"models":[" z-model ","","a-model","z-model"]}`)
	require.NoError(t, err)
	require.True(t, config.Enabled)
	require.Equal(t, []string{"a-model", "z-model"}, config.Models)
}

func TestParseRejectsInvalidJSONAndNonObject(t *testing.T) {
	_, err := Parse(`{invalid}`)
	require.Error(t, err)
	_, err = Parse(`[]`)
	require.Error(t, err)
}

func TestSerializeUsesStableEmptyModelsArray(t *testing.T) {
	serialized, err := Serialize(Config{Enabled: false})
	require.NoError(t, err)
	require.Equal(t, DefaultSerialized, serialized)
}
