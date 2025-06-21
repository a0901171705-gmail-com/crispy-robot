package interpreter_test

import (
	"os"
	"testing"

	"github.com/cli/cli/v2/hacceptance/interpreter"
	"github.com/cli/cli/v2/hacceptance/parser"
)

func TestAuthLogin(t *testing.T) {
	b, err := os.ReadFile("../parser/testdata/auth-login.txtar")
	if err != nil {
		t.Fatal(err)
	}

	parsedScript, err := parser.Parse(string(b))
	if err != nil {
		t.Fatal(err)
	}

	err = interpreter.Execute(t, parsedScript)
	if err != nil {
		t.Fatal(err)
	}
}
