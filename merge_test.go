package merge

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	r := require.New(t)

	expectedOut, err := os.ReadFile("_testdata/expected.go")
	r.NoError(err)

	config, b, err := Run("example/example-gomergetypes.yml")
	r.NoError(err)
	r.NotNil(config)
	r.Equal(string(expectedOut), string(b))
}
