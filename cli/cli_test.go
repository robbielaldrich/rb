package cli

import (
	"flag"
	"strings"
	"testing"
)

func TestEveryListedCommandBinds(t *testing.T) {
	for _, cmd := range commands {
		if bind(cmd, flag.NewFlagSet(cmd, flag.ContinueOnError)) == nil {
			t.Errorf("%s is listed but bind has no case for it", cmd)
		}
	}
}

func TestUsageDescribesEveryCommand(t *testing.T) {
	var out strings.Builder
	Usage(&out)

	for _, cmd := range commands {
		if !strings.Contains(out.String(), cmd) {
			t.Errorf("usage doesn't mention %s", cmd)
		}
	}

	for _, want := range []string{
		`  -out string        directory to write downloaded card data into (default "cards")`,
		`  -images            also download card images (default true)`,
		`  -concurrency int   number of concurrent image downloads (default 8)`,
		`  -all-printings         make a note per printing rather than per card`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("usage is missing:\n%s\ngot:\n%s", want, out.String())
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	err := Run([]string{"bogus"})
	if !IsUsage(err) {
		t.Fatalf("Run(bogus) = %v, want a usage error", err)
	}
	if !strings.Contains(err.Error(), `"bogus"`) {
		t.Errorf("error %q doesn't name the command", err)
	}
}

func TestNoCommand(t *testing.T) {
	if err := Run(nil); !IsUsage(err) {
		t.Fatalf("Run(nil) = %v, want a usage error", err)
	}
}

func TestUndefinedFlagIsNotAUsageError(t *testing.T) {
	err := Run([]string{"collect", "-nope"})
	if err == nil || IsUsage(err) {
		t.Fatalf("Run(collect -nope) = %v, want a parse failure", err)
	}
}
