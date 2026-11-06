package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestInvalidCommandsDoNotConnectToDatabase(t *testing.T) {
	for _, args := range [][]string{nil, {"set", "invented", "true", "--actor", "owner"}, {"set", "retired.enabled", "yes", "--actor", "owner"}, {"set", "retired.enabled", "true"}, {"set", "retired.enabled", "true", "--actor", " "}, {"list", "extra"}} {
		var output bytes.Buffer
		if err := run(args, &output); err == nil || strings.Contains(err.Error(), "database") {
			t.Fatalf("invalid command should fail before connecting: %v (%v)", args, err)
		}
	}
}
