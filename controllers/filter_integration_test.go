package controllers_test

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

func TestRequestLoggingFilterLogsNormalResponses(t *testing.T) {
	resetFake(t)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	do("GET", "/")
	do("GET", "/events/ev1")

	out := buf.String()
	for _, want := range []string{"[http] GET / -> 200", "[http] GET /events/ev1 -> 200"} {
		if !strings.Contains(out, want) {
			t.Errorf("want log line %q, got:\n%s", want, out)
		}
	}
}