package parser_test

import (
	"os"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/hacceptance/parser"
	"github.com/google/go-cmp/cmp"
)

func TestParseSingleInvocation(t *testing.T) {
	script := `gh auth login`
	parsedScript, err := parser.Parse(script)
	if err != nil {
		t.Fatal(err)
	}

	expectedScript := parser.Script{
		Interactions: []parser.Interaction{
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
		},
	}

	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
		t.Fatal(diff)
	}
}

func TestMultipleInvocations(t *testing.T) {
	script := heredoc.Doc(`
		gh auth login
		gh auth logout
		`)
	parsedScript, err := parser.Parse(script)
	if err != nil {
		t.Fatal(err)
	}

	expectedScript := parser.Script{
		Interactions: []parser.Interaction{
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "logout"}},
		},
	}

	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
		t.Fatal(diff)
	}
}

func TestSingleLineExpectationAfterInvocation(t *testing.T) {
	script := heredoc.Doc(`
	gh auth login
	---
	some expected output
	---
	`)
	parsedScript, err := parser.Parse(script)
	if err != nil {
		t.Fatal(err)
	}

	expectedScript := parser.Script{
		Interactions: []parser.Interaction{
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
			parser.Expectation{Content: "some expected output\n"},
		},
	}

	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
		t.Fatal(diff)
	}
}

func TestMultilineExpectationAfterInvocation(t *testing.T) {
	script := heredoc.Doc(`
	gh auth login
	---
	some expected output
	some more expected output
	---
	`)
	parsedScript, err := parser.Parse(script)
	if err != nil {
		t.Fatal(err)
	}

	expectedScript := parser.Script{
		Interactions: []parser.Interaction{
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
			parser.Expectation{Content: "some expected output\nsome more expected output\n"},
		},
	}

	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
		t.Fatal(diff)
	}
}

func TestMultipleInvocationsAndExpectations(t *testing.T) {
	script := heredoc.Doc(`
	gh auth login
	---
	some expected output
	---

	gh auth logout
	---
	some more expected output
	---
	`)
	parsedScript, err := parser.Parse(script)
	if err != nil {
		t.Fatal(err)
	}

	expectedScript := parser.Script{
		Interactions: []parser.Interaction{
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
			parser.Expectation{Content: "some expected output\n"},
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "logout"}},
			parser.Expectation{Content: "some more expected output\n"},
		},
	}

	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
		t.Fatal(diff)
	}
}

// TODO: unhappy case where there is no invocation before an action
// TODO: unhappy case where there is no expectation before an action
// TODO: case where Option has spaces in it

func TestSelectAction(t *testing.T) {
	script := heredoc.Doc(`
	gh auth login
	select Option
	`)
	parsedScript, err := parser.Parse(script)
	if err != nil {
		t.Fatal(err)
	}

	expectedScript := parser.Script{
		Interactions: []parser.Interaction{
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
			parser.Select{Option: "Option"},
		},
	}

	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
		t.Fatal(diff)
	}
}

// TODO: case where Say content has spaces in it

func TestSayAction(t *testing.T) {
	script := heredoc.Doc(`
	gh auth login
	say something
	`)
	parsedScript, err := parser.Parse(script)
	if err != nil {
		t.Fatal(err)
	}

	expectedScript := parser.Script{
		Interactions: []parser.Interaction{
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
			parser.Say{Text: "something"},
		},
	}

	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
		t.Fatal(diff)
	}
}

func TestBlankLinesAreIgnored(t *testing.T) {
	script := heredoc.Doc(`
		gh auth login

		gh auth logout
		`)
	parsedScript, err := parser.Parse(script)
	if err != nil {
		t.Fatal(err)
	}

	expectedScript := parser.Script{
		Interactions: []parser.Interaction{
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "logout"}},
		},
	}

	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
		t.Fatal(diff)
	}
}

func TestParseFromFile(t *testing.T) {
	b, err := os.ReadFile("./testdata/auth-login.txtar")
	if err != nil {
		t.Fatal(err)
	}

	parsedScript, err := parser.Parse(string(b))
	if err != nil {
		t.Fatal(err)
	}

	expectedFirstExpectation := heredoc.Doc(`? Where do you use GitHub?  [Use arrows to move, type to filter]
> GitHub.com
  Other
`)

	expectedSecondExpectation := heredoc.Doc(`? Where do you use GitHub? Other
? Hostname: `)

	expectedThirdExpectation := heredoc.Doc(`? Hostname: my.ghes.com
? What is your preferred protocol for Git operations on this host?  [Use arrows to move, type to filter]
> HTTPS
  SSH
`)

	expectedFourthExpectation := heredoc.Doc(`? Authenticate Git with your GitHub credentials? (Y/n) `)

	expectedScript := parser.Script{
		Interactions: []parser.Interaction{
			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
			parser.Expectation{Content: expectedFirstExpectation},
			parser.Select{Option: "Other"},
			parser.Expectation{Content: expectedSecondExpectation},
			parser.Say{Text: "my.ghes.com"},
			parser.Expectation{Content: expectedThirdExpectation},
			parser.Select{Option: "HTTPS"},
			parser.Expectation{Content: expectedFourthExpectation},
		},
	}

	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
		t.Fatal(diff)
	}
}

// func TestLinesWithWhitespaceCharsAreAnError(t *testing.T) {
// 	script := `gh auth login\n   \ngh auth logout`
// 	parsedScript, err := parser.Parse(script)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	expectedScript := parser.Script{
// 		Interactions: []parser.Interaction{
// 			parser.Invocation{Cmd: "gh", Args: []string{"auth", "login"}},
// 			parser.Invocation{Cmd: "gh", Args: []string{"auth", "logout"}},
// 		},
// 	}

// 	if diff := cmp.Diff(expectedScript, parsedScript); diff != "" {
// 		t.Fatal(diff)
// 	}
// }
