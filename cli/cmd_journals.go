package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) journalsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journals <query>",
		Short: "Search DOAJ journals",
		Long: `Search DOAJ journals by full-text query.

Supports Lucene syntax: publisher:MDPI, bibjson.title:biology, etc.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("searching journals for %q...", args[0])
			journals, err := a.client.SearchJournals(cmd.Context(), args[0], n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(journals, len(journals))
		},
	}
	return cmd
}

func (a *App) journalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journal <issn>",
		Short: "Fetch a single DOAJ journal by ISSN",
		Long: `Fetch and display a single journal from DOAJ by its ISSN.

The ISSN may be supplied with or without the hyphen (e.g. 1932-6203 or 19326203).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching journal %s...", args[0])
			j, err := a.client.GetJournal(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(j)
		},
	}
	return cmd
}
