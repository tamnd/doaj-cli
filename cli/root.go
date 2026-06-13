// Package cli builds the doaj command tree on top of the doaj library.
package cli

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/tamnd/doaj-cli/doaj"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// exit codes.
const (
	exitError  = 1
	exitUsage  = 2
	exitNoData = 3
)

// App holds shared state threaded through every command.
type App struct {
	client *doaj.Client
	cfg    doaj.Config

	output   string
	fields   []string
	noHeader bool
	template string
	limit    int
	quiet    bool
}

// Root builds the root command and its subtree.
func Root() *cobra.Command {
	app := &App{cfg: doaj.DefaultConfig()}

	root := &cobra.Command{
		Use:   "doaj",
		Short: "Search DOAJ open access journals and articles",
		Long: `doaj reads the Directory of Open Access Journals (DOAJ) through its
public REST API v3. No API key is required. It returns records as
table, JSON, JSONL, CSV, TSV, or URLs.

doaj is an independent tool and is not affiliated with DOAJ or IS4OA.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return app.setup()
		},
	}

	pf := root.PersistentFlags()
	pf.StringVarP(&app.output, "output", "o", "auto", "output: table|json|jsonl|csv|tsv|url|raw (auto=table on TTY, jsonl piped)")
	pf.StringSliceVar(&app.fields, "fields", nil, "comma-separated columns to include")
	pf.BoolVar(&app.noHeader, "no-header", false, "omit the header row in table/csv/tsv")
	pf.StringVar(&app.template, "template", "", "Go text/template applied per record")
	pf.IntVarP(&app.limit, "limit", "n", 0, "limit number of records (0 = command default)")
	pf.BoolVarP(&app.quiet, "quiet", "q", false, "suppress progress on stderr")

	pf.DurationVar(&app.cfg.Rate, "delay", app.cfg.Rate, "minimum spacing between requests")
	pf.DurationVar(&app.cfg.Timeout, "timeout", app.cfg.Timeout, "per-request timeout")
	pf.IntVar(&app.cfg.Retries, "retries", app.cfg.Retries, "retry attempts on 429/5xx")
	pf.StringVar(&app.cfg.UserAgent, "user-agent", app.cfg.UserAgent, "User-Agent sent with each request")

	root.AddCommand(
		app.searchCmd(),
		app.journalsCmd(),
		app.journalCmd(),
		app.articleCmd(),
		app.topCmd(),
		newVersionCmd(),
	)
	return root
}

func (a *App) setup() error {
	if a.output == "" || a.output == "auto" {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			a.output = string(FormatTable)
		} else {
			a.output = string(FormatJSONL)
		}
	}
	if !Format(a.output).Valid() {
		return codeError(exitUsage, fmt.Errorf("unknown output format %q", a.output))
	}
	a.client = doaj.NewClient(a.cfg)
	return nil
}

func (a *App) render(records any) error {
	r := NewRenderer(os.Stdout, Format(a.output), a.fields, a.noHeader, a.template)
	return r.Render(records)
}

func (a *App) renderOrEmpty(records any, n int) error {
	if err := a.render(records); err != nil {
		return err
	}
	if n == 0 {
		return codeError(exitNoData, nil)
	}
	return nil
}

func (a *App) progressf(format string, args ...any) {
	if a.quiet {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func (a *App) effectiveLimit(def int) int {
	if a.limit > 0 {
		return a.limit
	}
	return def
}
