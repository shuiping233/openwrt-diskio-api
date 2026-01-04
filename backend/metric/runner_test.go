package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunnerRun(t *testing.T) {
	runner := CommandRunner{}

	result, err := runner.Run("echo", "hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", result)

	result, err = runner.Run("")
	assert.Error(t, err)

	result, err = runner.Run("cat", "/path/notExist")
	assert.Error(t, err)
}
