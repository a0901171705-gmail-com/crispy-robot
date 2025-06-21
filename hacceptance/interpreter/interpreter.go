package interpreter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Netflix/go-expect"
	"github.com/acarl005/stripansi"
	"github.com/cli/cli/v2/hacceptance/parser"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

type interpreter struct {
	cmd     *exec.Cmd
	console *expect.Console
}

// TODO: figure out a way to buffer the previous lines when doing expectations

func Execute(t *testing.T, script parser.Script) error {
	state := interpreter{}

	for _, i := range script.Interactions {
		fmt.Printf("%+v\n", i)

		switch interaction := i.(type) {
		case parser.Invocation:
			// Create a PTY and hook up a virtual terminal emulator
			ptm, pts, err := pty.Open()
			require.NoError(t, err)

			term := vt10x.New(vt10x.WithWriter(pts))

			// Create a console via Expect that allows scripting against the terminal
			consoleOpts := []expect.ConsoleOpt{
				expect.WithStdin(ptm),
				expect.WithStdout(term),
				expect.WithCloser(ptm, pts),
				failOnExpectError(t),
				failOnSendError(t),
				expect.WithDefaultTimeout(time.Second * 2),
			}

			console, err := expect.NewConsole(consoleOpts...)
			require.NoError(t, err)
			t.Cleanup(func() { testCloser(t, console) })

			// Let's make sure that ascinema is available
			path, err := exec.LookPath("asciinema")
			require.NoError(t, err, "asciinema must be installed to record tests")

			// Pick a random name for the recording inside the temp dir
			recordingName := randomString(5)
			recordingPath := fmt.Sprintf("%s/%s", os.TempDir(), recordingName)
			t.Cleanup(func() {
				t.Logf("Check out the recording by running `asciinema play %s`", recordingPath)
			})

			// Create the command telling asciinema to use the binary location with the given args
			cmd := exec.Command(
				path, "rec",
				"-c", fmt.Sprintf("%s %s", interaction.Cmd, strings.Join(interaction.Args, " ")),
				"-q", recordingPath)

			// And here we need to do some magic because asciinema expects to read directly from /dev/tty,
			// rather than the stdout fd, so we need to tell the kernel that the controlling tty for this process
			// (that is pointed to by /dev/tty) is the console PTY we created above. To do that we set Ctty to the FD
			// of our PTY, and to ensure that always has the same FD number we add it as an extra file.
			//
			// Finally, a process can only have a controlling TTY set if it is a session leader, so we set Setsid
			// to true.
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setsid:  true,
				Setctty: true,
				Ctty:    3,
			}
			cmd.ExtraFiles = append(cmd.ExtraFiles, console.Tty())
			cmd.Stdin = console.Tty()
			cmd.Stdout = console.Tty()
			cmd.Stderr = console.Tty()
			cmd.Env = append(os.Environ(), "CLICOLOR=0")

			// Start the command
			// TODO: We currently provide no method for asserting that the command has exited,
			// or cleaning it up if it hasn't, so let's do that soon.
			require.NoError(t, cmd.Start())

			state.cmd = cmd
			state.console = console
		case parser.Expectation:
			// Error checks are provided by failOnExpectError
			_, _ = state.console.Expect(ANSIStrippedString(interaction.Content))
		case parser.Action:
			switch action := interaction.(type) {
			case parser.Select:
				// Error checks are provided by failOnSendError
				// TODO: confirm that newline is the same as hitting enter.
				_, _ = state.console.SendLine(action.Option)
				// _, _ = state.console.Send(string(KeyEnter))
			case parser.Say:
				// Error checks are provided by failOnSendError
				_, _ = state.console.SendLine(action.Text)
			}
		}
	}

	require.NoError(t, state.cmd.Process.Kill())

	return nil
}

// failOnExpectError adds an observer that will fail the test in a standardised way
// if any expectation on the command output fails, without requiring an explicit
// assertion.
//
// Use WithRelaxedIO to disable this behaviour.
func failOnExpectError(t testing.TB) expect.ConsoleOpt {
	t.Helper()
	return expect.WithExpectObserver(
		func(matchers []expect.Matcher, buf string, err error) {
			t.Helper()

			if err == nil {
				return
			}

			if len(matchers) == 0 {
				t.Fatalf("Error occurred while matching %q: %s\n", buf, err)
			}

			var criteria []string
			for _, matcher := range matchers {
				criteria = append(criteria, fmt.Sprintf("%q", matcher.Criteria()))
			}
			t.Fatalf("Failed to find [%s] in %q: %s\n", strings.Join(criteria, ", "), buf, err)
		},
	)
}

// failOnSendError adds an observer that will fail the test in a standardised way
// if any sending of input fails, without requiring an explicit assertion.
//
// Use WithRelaxedIO to disable this behaviour.
func failOnSendError(t testing.TB) expect.ConsoleOpt {
	t.Helper()
	return expect.WithSendObserver(
		func(msg string, n int, err error) {
			t.Helper()

			if err != nil {
				t.Fatalf("Failed to send %q: %s\n", msg, err)
			}
			if len(msg) != n {
				t.Fatalf("Only sent %d of %d bytes for %q\n", n, len(msg), msg)
			}
		},
	)
}

// testCloser is a helper to fail the test if a Closer fails to close.
func testCloser(t testing.TB, closer io.Closer) {
	t.Helper()
	if err := closer.Close(); err != nil {
		t.Errorf("Close failed: %s", err)
	}
}

// ansiStrippedStringMatcher fulfills the Matcher interface to match strings against a given
// bytes.Buffer, with ANSI escape codes stripped.
type ansiStrippedStringMatcher struct {
	str string
}

func (sm *ansiStrippedStringMatcher) Match(v interface{}) bool {
	buf, ok := v.(*bytes.Buffer)
	if !ok {
		return false
	}

	s := buf.String()
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = stripansi.Strip(s)

	return strings.Contains(s, sm.str)
}

func (sm *ansiStrippedStringMatcher) Criteria() interface{} {
	return sm.str
}

// ANSIStrippedString returns an expect.ExpectOpt that checks whether the text is included
// in the console buffer after ANSI escape codes have been stripped.
func ANSIStrippedString(str string) expect.ExpectOpt {
	return func(opts *expect.ExpectOpts) error {
		opts.Matchers = append(opts.Matchers, &ansiStrippedStringMatcher{
			str: str,
		})

		return nil
	}
}

// Key represents a keypress in rune form.
// Copied directly from terminal/sequences.go with an extra type for funsies.
type Key rune

// SA9004
//
//nolint:revive,staticcheck // The names are exactly what they are and the type is the same
const (
	KeyArrowLeft       Key = '\x02'
	KeyArrowRight          = '\x06'
	KeyArrowUp             = '\x10'
	KeyArrowDown           = '\x0e'
	KeySpace               = ' '
	KeyEnter               = '\r'
	KeyBackspace           = '\b'
	KeyDelete              = '\x7f'
	KeyInterrupt           = '\x03'
	KeyEndTransmission     = '\x04'
	KeyEscape              = '\x1b'
	// KeyDeleteWord is Ctrl+W.
	KeyDeleteWord = '\x17'
	// KeyDeleteLine is Ctrl+X.
	KeyDeleteLine    = '\x18'
	SpecialKeyHome   = '\x01'
	SpecialKeyEnd    = '\x11'
	SpecialKeyDelete = '\x12'
	IgnoreKey        = '\000'
	KeyTab           = '\t'
)

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	//nolint:gosec // this doesn't need to be cryptographically secure
	seededRand := rand.New(
		rand.NewSource(uint64(time.Now().UnixNano())))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// ? Where do you use GitHub?  [Use arrows to move, type to filter]
// > GitHub.com
//   Other

// ? Where do you use GitHub?  [Use arrows to move, type to filter]\x1b[0m\r\n\x1b[0;1;36m> GitHub.com\x1b[0m\r\n\x1b[0;39m  Other\x1b[0m\r\n\x1b7\x1b[1A\x1b[0G\x1b[1A\x1b[0G
