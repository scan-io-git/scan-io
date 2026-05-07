package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
)

func setupExecuteTestGlobals(ci bool) {
	mode := "user"
	if ci {
		mode = "CI"
	}
	AppConfig = &config.Config{}
	AppConfig.Scanio.Mode = mode
	Logger = hclog.NewNullLogger()
}

func captureStderr(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestHandleExecuteError_RawError_PrintsToStderrAndReturns1(t *testing.T) {
	setupExecuteTestGlobals(false)
	var code int
	stderr := captureStderr(func() {
		code = handleExecuteError(fmt.Errorf("some cobra error"))
	})
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "some cobra error")
}

func TestHandleExecuteError_CommandError_ReturnsExitCodeAndPrintsMessage(t *testing.T) {
	setupExecuteTestGlobals(false)
	cmdErr := errors.NewCommandError(nil, nil, fmt.Errorf("plugin failed"), 3)
	var code int
	stderr := captureStderr(func() {
		code = handleExecuteError(cmdErr)
	})
	assert.Equal(t, 3, code)
	assert.Contains(t, stderr, "plugin failed")
}

func TestHandleExecuteError_CIMode_EmitsJSONAndReturns1(t *testing.T) {
	setupExecuteTestGlobals(true)
	var code int
	stdout := captureStdout(func() {
		code = handleExecuteError(fmt.Errorf("ci raw error"))
	})
	assert.Equal(t, 1, code)
	require.Contains(t, stdout, "FAILED")
	assert.Contains(t, stdout, "ci raw error")
}
