package user

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/superduck-ai/yourbatis"
)

func captureEnvironmentDebugOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	t.Setenv(yourbatis.DebugEnvironmentVariable, "1")

	output := &bytes.Buffer{}
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	log.SetOutput(output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})
	return output
}

func assertEnvironmentDebugOutput(
	t *testing.T,
	output string,
	expected []string,
	forbidden []string,
) {
	t.Helper()
	for _, value := range expected {
		if !strings.Contains(output, value) {
			t.Fatalf("environment debug output does not contain %q:\n%s", value, output)
		}
	}
	for _, value := range forbidden {
		if strings.Contains(output, value) {
			t.Fatalf("environment debug output unexpectedly contains %q:\n%s", value, output)
		}
	}
}
