package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search DOAJ articles",
		Long: `Search DOAJ articles by full-text query.

Supports Lucene syntax: author:"Smith", bibjson.title:climate, etc.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("searching articles for %q...", args[0])
			arts, err := a.client.SearchArticles(cmd.Context(), args[0], n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(arts, len(arts))
		},
	}
	return cmd
}

func (a *App) topCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Browse top DOAJ journals",
		Long:  `Browse the first page of journals indexed in DOAJ.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching top %d journals...", n)
			journals, err := a.client.SearchJournals(cmd.Context(), "*", n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(journals, len(journals))
		},
	}
	return cmd
}
