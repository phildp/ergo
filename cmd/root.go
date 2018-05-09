package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dbaltas/ergo/github"
	"github.com/dbaltas/ergo/repo"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	repoURL               string
	directory             string
	skipFetch             bool
	baseBranch            string
	branchesString        string
	releaseBranchesString string
	branches              []string
	releaseBranches       []string
	organizationName      string
	repoName              string
	releaseRepo           string

	gc      *github.Client
	gitRepo *repo.Repo
)

var rootCmd = &cobra.Command{
	Use:   "ergo",
	Short: "ergo is a tool that aims to help the daily developer workflow",
	Long: `Ergo aids to compare multiple branches.
On cases where deployment is done by pushing on a git branch:
* it can draft a github release,
* deploy on multiple branches and update the release notes with the time of release
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hola! type `ergo help`")
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err error

		gitRepo, err = initializeRepo()
		if err != nil {
			fmt.Printf("get repo: %v\n", err)
			os.Exit(1)
		}

		gc, err = github.NewClient(context.Background(), viper.GetString("github.access-token"), organizationName, releaseRepo)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&repoURL, "repoUrl", "", "git repo Url. ssh and https supported")
	rootCmd.PersistentFlags().StringVar(&directory, "directory", ".", "Location to store or retrieve from the repo")
	rootCmd.PersistentFlags().BoolVar(&skipFetch, "skipFetch", false, "Skip fetch. When set you may not be up to date with remote")

	rootCmd.PersistentFlags().StringVar(&branchesString, "branches", "", "Comma separated list of branches")
	rootCmd.PersistentFlags().StringVar(&baseBranch, "baseBranch", "", "Base branch for the comparison.")
}

func initConfig() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	viper.AddConfigPath(home)
	viper.SetConfigName(".ergo")
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("error reading config file: %v\n", err)
		os.Exit(1)
	}
	if baseBranch == "" {
		baseBranch = viper.GetString("generic.base-branch")
	}
}

func initializeRepo() (*repo.Repo, error) {
	r := repo.New(repoURL, directory, viper.GetString("generic.remote"))
	repository, err := r.LoadOrClone(skipFetch)
	if err != nil {
		fmt.Printf("Error loading repo:%s\n", err)
		return nil, err
	}

	rmt, err := repository.Remote(viper.GetString("generic.remote"))

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	parts := strings.Split(rmt.Config().URLs[0], "/")
	repoName = strings.TrimSuffix(parts[len(parts)-1], ".git")
	organizationName = parts[len(parts)-2]
	// if remote is set by ssh instead of https
	if strings.Contains(organizationName, ":") {
		organizationName = organizationName[strings.LastIndex(organizationName, ":")+1:]
	}

	releaseBranchesString = branchesString
	if branchesString == "" {
		branchesString = viper.GetString(fmt.Sprintf("repos.%s.status-branches", repoName))
		releaseBranchesString = viper.GetString(fmt.Sprintf("repos.%s.release-branches", repoName))
	}

	if branchesString == "" {
		branchesString = viper.GetString("generic.status-branches")
	}

	if releaseBranchesString == "" {
		releaseBranchesString = viper.GetString("generic.release-branches")
	}

	branches = strings.Split(branchesString, ",")
	releaseBranches = strings.Split(releaseBranchesString, ",")

	fmt.Println(repoName)
	if strings.Contains(viper.GetString("generic.release-repos"), repoName) {
		releaseRepo = repoName
	}

	return r, nil
}

// Execute entry point for commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
