package autolink

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/browser"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		output       listOptions
		wantErr      bool
		wantExporter bool
		errMsg       string
	}{
		{
			name:   "no argument",
			input:  "",
			output: listOptions{},
		},
		{
			name:   "web flag",
			input:  "--web",
			output: listOptions{WebMode: true},
		},
		{
			name:         "json flag",
			input:        "--json id",
			output:       listOptions{},
			wantExporter: true,
		},
		{
			name:    "invalid json flag",
			input:   "--json invalid",
			wantErr: true,
			errMsg: heredoc.Doc(`
				Unknown JSON field: "invalid"
				Available fields:
				  id
				  isAlphanumeric
				  keyPrefix
				  urlTemplate`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}
			f.HttpClient = func() (*http.Client, error) {
				return &http.Client{}, nil
			}

			argv, err := shlex.Split(tt.input)
			require.NoError(t, err)

			var gotOpts *listOptions
			cmd := newCmdList(f, func(opts *listOptions) error {
				gotOpts = opts
				return nil
			})

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantErr {
				require.EqualError(t, err, tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.output.WebMode, gotOpts.WebMode)
			assert.Equal(t, tt.wantExporter, gotOpts.Exporter != nil)
		})
	}
}

type mockAutoLinkGetter struct {
	Response []autolink
}

func (m *mockAutoLinkGetter) Get(repo ghrepo.Interface) ([]autolink, error) {
	return m.Response, nil
}

type mockAutoLinkGetterError struct {
	err error
}

func (me *mockAutoLinkGetterError) Get(repo ghrepo.Interface) ([]autolink, error) {
	return nil, me.err
}

func TestListRun(t *testing.T) {
	tests := []struct {
		name       string
		opts       *listOptions
		isTTY      bool
		response   []autolink
		wantStdout string
		wantStderr string
		wantErr    bool
	}{
		{
			name:  "list tty",
			opts:  &listOptions{},
			isTTY: true,
			response: []autolink{
				{
					ID:             1,
					KeyPrefix:      "TICKET-",
					URLTemplate:    "https://example.com/TICKET?query=<num>",
					IsAlphanumeric: true,
				},
				{
					ID:             2,
					KeyPrefix:      "STORY-",
					URLTemplate:    "https://example.com/STORY?id=<num>",
					IsAlphanumeric: false,
				},
			},
			wantStdout: heredoc.Doc(`

				Showing 2 autolink references in OWNER/REPO

				ID  KEY PREFIX  URL TEMPLATE                            ALPHANUMERIC
				1   TICKET-     https://example.com/TICKET?query=<num>  true
				2   STORY-      https://example.com/STORY?id=<num>      false
			`),
			wantStderr: "",
		},
		{
			name: "list json",
			opts: &listOptions{
				Exporter: func() cmdutil.Exporter {
					exporter := cmdutil.NewJSONExporter()
					exporter.SetFields([]string{"id"})
					return exporter
				}(),
			},
			isTTY: true,
			response: []autolink{
				{
					ID:             1,
					KeyPrefix:      "TICKET-",
					URLTemplate:    "https://example.com/TICKET?query=<num>",
					IsAlphanumeric: true,
				},
				{
					ID:             2,
					KeyPrefix:      "STORY-",
					URLTemplate:    "https://example.com/STORY?id=<num>",
					IsAlphanumeric: false,
				},
			},
			wantStdout: "[{\"id\":1},{\"id\":2}]\n",
			wantStderr: "",
		},
		{
			name:  "list non-tty",
			opts:  &listOptions{},
			isTTY: false,
			response: []autolink{
				{
					ID:             1,
					KeyPrefix:      "TICKET-",
					URLTemplate:    "https://example.com/TICKET?query=<num>",
					IsAlphanumeric: true,
				},
				{
					ID:             2,
					KeyPrefix:      "STORY-",
					URLTemplate:    "https://example.com/STORY?id=<num>",
					IsAlphanumeric: false,
				},
			},
			wantStdout: heredoc.Doc(`
				1	TICKET-	https://example.com/TICKET?query=<num>	true
				2	STORY-	https://example.com/STORY?id=<num>	false
			`),
			wantStderr: "",
		},

		{
			name:       "no results",
			opts:       &listOptions{},
			isTTY:      true,
			response:   []autolink{},
			wantStdout: "",
			wantStderr: "",
			wantErr:    true,
		},
		{
			name:       "web mode",
			isTTY:      true,
			opts:       &listOptions{WebMode: true},
			wantStderr: "Opening https://github.com/OWNER/REPO/settings/key_links in your browser.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.isTTY)
			ios.SetStdinTTY(tt.isTTY)
			ios.SetStderrTTY(tt.isTTY)

			opts := tt.opts
			opts.IO = ios
			opts.Browser = &browser.Stub{}

			opts.IO = ios
			opts.BaseRepo = func() (ghrepo.Interface, error) { return ghrepo.New("OWNER", "REPO"), nil }

			if tt.wantErr {
				opts.AutolinkClient = &mockAutoLinkGetterError{err: fmt.Errorf("mock error")}
				err := listRun(opts)
				require.Error(t, err)
			} else {
				opts.AutolinkClient = &mockAutoLinkGetter{Response: tt.response}
				err := listRun(opts)
				require.NoError(t, err)
				assert.Equal(t, tt.wantStdout, stdout.String())
			}

			if tt.wantStderr != "" {
				assert.Equal(t, tt.wantStderr, stderr.String())
			}
		})
	}
}
