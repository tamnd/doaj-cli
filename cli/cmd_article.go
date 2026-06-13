package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) articleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "article <id>",
		Short: "Fetch a single DOAJ article by ID",
		Long:  `Fetch and display a single article from DOAJ by its internal ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching article %s...", args[0])
			art, err := a.client.GetArticle(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(art)
		},
	}
	return cmd
}
