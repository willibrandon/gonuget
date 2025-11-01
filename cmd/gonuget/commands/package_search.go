package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
	"github.com/willibrandon/gonuget/core"
)

// PackageSearchOptions holds the configuration for the package search command.
type PackageSearchOptions struct {
	Source     string
	Format     string
	Take       int
	Skip       int
	Prerelease bool
}

// NewPackageSearchCommand creates the 'package search' subcommand.
func NewPackageSearchCommand() *cobra.Command {
	opts := &PackageSearchOptions{}

	cmd := &cobra.Command{
		Use:   "search <SEARCH_TERM>",
		Short: "Search for NuGet packages",
		Long: `Search for NuGet packages in configured package sources.

This command searches for packages matching the search term using the NuGet V3 search API.
Results can be paginated using --skip and --take flags.
Output can be formatted as console (human-readable) or JSON.

Examples:
  gonuget package search Newtonsoft
  gonuget package search Serilog --take 10
  gonuget package search EntityFramework --format json
  gonuget package search AspNetCore --prerelease`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			searchTerm := args[0]
			return runPackageSearch(cmd.Context(), searchTerm, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Source, "source", "s", "", "Package source to search")
	cmd.Flags().StringVar(&opts.Format, "format", "console", "Output format: console or json")
	cmd.Flags().IntVar(&opts.Take, "take", 20, "Number of results to return")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "Number of results to skip (for pagination)")
	cmd.Flags().BoolVar(&opts.Prerelease, "prerelease", false, "Include prerelease packages")

	return cmd
}

// runPackageSearch implements the package search command logic.
func runPackageSearch(ctx context.Context, searchTerm string, opts *PackageSearchOptions) error {
	start := time.Now()

	// Create a client with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Determine source
	source := opts.Source
	if source == "" {
		// Load first enabled source from config with fallback to defaults
		projectDir, err := os.Getwd()
		if err != nil {
			projectDir = "."
		}

		sources := config.GetEnabledSourcesOrDefault(projectDir)
		if len(sources) > 0 {
			source = sources[0].Value
		} else {
			return fmt.Errorf("no package sources configured")
		}
	}

	// Track sources for JSON output
	searchedSources := []string{source}

	// Create NuGet client with repository manager
	repoManager := core.NewRepositoryManager()

	// Add the source repository
	repo := core.NewSourceRepository(core.RepositoryConfig{
		SourceURL: source,
		Name:      "default",
	})
	if err := repoManager.AddRepository(repo); err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}

	client := core.NewClient(core.ClientConfig{
		RepositoryManager: repoManager,
	})

	// Perform search across all repositories
	searchOpts := core.SearchOptions{
		Skip:              opts.Skip,
		Take:              opts.Take,
		IncludePrerelease: opts.Prerelease,
	}

	resultsMap, err := client.SearchPackages(ctx, searchTerm, searchOpts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Flatten results from all sources
	var allResults []core.SearchResult
	for _, results := range resultsMap {
		allResults = append(allResults, results...)
	}

	// Output based on format
	if opts.Format == "json" {
		return outputSearchResultsJSON(searchTerm, searchedSources, allResults, start)
	}

	return outputSearchResultsConsole(searchTerm, source, allResults)
}

// outputSearchResultsConsole outputs search results in human-readable format
func outputSearchResultsConsole(searchTerm, source string, results []core.SearchResult) error {
	fmt.Printf("Searching for '%s' in source: %s\n", searchTerm, filepath.Base(source))
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("No packages found matching the search criteria.")
		return nil
	}

	for _, pkg := range results {
		fmt.Printf("> %s\n", pkg.ID)
		if pkg.Description != "" {
			fmt.Printf("  %s\n", pkg.Description)
		}
		fmt.Printf("  Latest: %s | Downloads: %d\n", pkg.Version, pkg.TotalDownloads)
		fmt.Println()
	}

	fmt.Printf("Showing %d results\n", len(results))

	return nil
}

// outputSearchResultsJSON outputs search results in JSON format matching schema
func outputSearchResultsJSON(searchTerm string, sources []string, results []core.SearchResult, start time.Time) error {
	jsonOutput := output.NewPackageSearchOutput(searchTerm, sources, start)

	// Convert core.SearchResult to output.SearchResult
	for _, pkg := range results {
		// Convert authors array to comma-separated string
		authorsStr := ""
		if len(pkg.Authors) > 0 {
			authorsStr = pkg.Authors[0]
			for i := 1; i < len(pkg.Authors); i++ {
				authorsStr += ", " + pkg.Authors[i]
			}
		}

		jsonOutput.Items = append(jsonOutput.Items, output.SearchResult{
			ID:             pkg.ID,
			Version:        pkg.Version,
			Description:    pkg.Description,
			Authors:        authorsStr,
			TotalDownloads: pkg.TotalDownloads,
			Verified:       pkg.Verified,
			IconURL:        pkg.IconURL,
			Tags:           pkg.Tags,
		})
	}

	// Set total count (Note: For now, same as items length. In future, this could be total available results)
	jsonOutput.Total = len(results)

	// Update elapsed time
	jsonOutput.ElapsedMs = output.MeasureElapsed(start)

	// Write JSON to stdout (VR-019: empty results return exit code 0 with valid JSON)
	return output.WriteJSON(os.Stdout, jsonOutput)
}

// init registers the package search subcommand with the package parent command
func init() {
	packageCmd := GetPackageCommand()
	packageCmd.AddCommand(NewPackageSearchCommand())
}
