package cmd

import (
	"context"
	"crypto/sha1"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/rs/xid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

var (
	accessToken string
	branch      string
	repo        string
	owner       string
	commit      string
	email       string
	pathFormat  string
	basePath    string
)

var u2gCmd = &cobra.Command{
	Use:   "u2g",
	Short: "upload file to github",
	Long:  `upload file to github,need github token config. use 'kt u2g -h' find help`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken = viper.GetString("github.token")
		if accessToken == "" {
			return errors.New("need github.token in config file")
		}

		f, err := os.Open(basePath)
		if err != nil {
			return err
		}

		fi, err := f.Stat()
		if err != nil {
			return err
		}

		client := github.NewClient(
			oauth2.NewClient(
				context.Background(),
				oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}),
			),
		)

		// Upload file or dir
		filePaths := map[string]string{}
		if fi.IsDir() {
			err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}

				githubFilePath, err := upload(client, path)
				if err == nil {
					filePaths[info.Name()] = githubFilePath
				}

				return err
			})

			if err != nil {
				return err
			}
		} else {
			githubFilePath, err := upload(client, basePath)
			if err != nil {
				return err
			}

			filePaths[fi.Name()] = githubFilePath
		}

		log.Printf("%v", filePaths)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(u2gCmd)

	u2gCmd.Flags().StringVarP(&basePath, "path", "p", "./u2g", "file or dir path")
	u2gCmd.Flags().StringVarP(&pathFormat, "format", "f", "20060102", "dir datetime format")
	u2gCmd.Flags().StringVarP(&email, "emial", "e", "kazma233@outlook.com", "github commit emial")
	u2gCmd.Flags().StringVarP(&commit, "commit", "c", "upload file via go client", "commit message")
	u2gCmd.Flags().StringVarP(&owner, "owner", "o", "kazma233", "repo owner")
	u2gCmd.Flags().StringVarP(&repo, "repo", "r", "static", "repo name")
	u2gCmd.Flags().StringVarP(&branch, "branch", "b", "master", "branch")
}

func upload(client *github.Client, uploadPath string) (string, error) {
	fileType := ""
	if strings.Contains(uploadPath, ".") {
		nameSplit := strings.Split(uploadPath, ".")
		fileType = nameSplit[len(nameSplit)-1]
	}

	file, err := os.OpenFile(uploadPath, os.O_RDONLY, 0755)
	if err != nil {
		return "", err
	}

	defer file.Close()

	fileByte, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	_sha := sha1.New()
	_, err = _sha.Write(fileByte)
	if err != nil {
		return "", err
	}

	time.Now().Local()
	githubPath := time.Now().Format(pathFormat) + "/" + xid.New().String() + "." + fileType
	date := time.Now()
	_, _, err = client.Repositories.CreateFile(context.Background(), owner, repo, githubPath, &github.RepositoryContentFileOptions{
		Message: github.String(commit),
		Content: fileByte,
		Branch:  github.String(branch),
		Author: &github.CommitAuthor{
			Date:  &date,
			Name:  github.String(owner),
			Email: github.String(email),
		},
		Committer: &github.CommitAuthor{
			Date:  &date,
			Name:  github.String(owner),
			Email: github.String(email),
		},
		SHA: github.String(string(_sha.Sum(nil))),
	})

	return "https://raw.githubusercontent.com/" + owner + "/" + repo + "/" + branch + "/" + githubPath, err
}
