package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/safeexec"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const APIEndpoint = "https://api.github.com/repos/"

type Release struct {
	TagName     string    `json:"tag_name"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
}

type Commits struct {
	SHA    string          `json:"sha"`
	NodeID string          `json:"node_id"`
	Url    string          `json:"html_url"`
	Parent []ParentCommits `json:"parents"`
}

// ParentCommits sub-structure of Commits
type ParentCommits struct {
	Sha     string `json:"sha"`
	Url     string `json:"url"`
	HtmlUrl string `json:"html_url"`
}

type Runs struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type Workflows struct {
	WorkflowsRuns []Runs `json:"workflow_runs"`
}

func main() {

	username := "Iltwats"
	repoName := "react-template"
	releaseURL := fmt.Sprintf(APIEndpoint+"%s/%s/releases", username, repoName)
	releaseData, err := getReleases(releaseURL)
	fmt.Println("Fetching all the release tags...")
	if err != nil {
		log.Fatalln(err)
	}
	// extract all the tags from commit json response
	var tags []string
	for _, val := range releaseData {
		tags = append(tags, val.TagName)
	}
	tagSelectedByUser := tags[0]
	userRepoConsumedTag := tags[1]  // TODO fetch from API
	isUserRepoStackConsumed := true // TODO fetch from API
	fmt.Printf("Current version of the repository: %s\n", userRepoConsumedTag)
	message := fmt.Sprintf("Upgrading this repository from version %s to version %s", userRepoConsumedTag, tagSelectedByUser)
	fmt.Println(message)
	var usesFile = "stack-init"
	var fileName = fmt.Sprintf("%s@%s.yml", usesFile, tagSelectedByUser)
	if isUserRepoStackConsumed {
		var fileUrl = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/.github/workflows/%s", username, repoName, fileName)
		if downloadTheWorkflowFile(fileName, fileUrl) {
			//MoveFile(fileName)
			//doGitOperationsForWorkflowFile(fileName)
			//errB := CheckoutBranch(fileName)
			//if errB != nil {
			//	log.Fatalln(errB)
			//}
			//errP := pushTheBranch(fileName)
			//if errP != nil {
			//	log.Fatalln(errP)
			//}
			//err := RunWorkflow(fileName)
			//if err != nil {
			//	log.Fatalln(err)
			//}
			pathName := getNames()
			fileUrl := fmt.Sprintf("https://api.github.com/repos%s/actions/workflows/%s/runs", pathName, fileName)
			fmt.Println(fileUrl)
			workflowStatsCheck(fileUrl)
		}
	}
}

func workflowStatsCheck(url string) {
	opt, er := getWorkflowRunStats(url)
	if er != nil {
		log.Fatalln(er)
	}
	status := opt.WorkflowsRuns[0].Status
	fmt.Printf("Waiting for workflow run to be completed, current status -> %s\n", status)

	//time.AfterFunc(59*time.Second, workflowStatsCheck(url))
}

func loopCase() {
	for {
		workflowStatsCheck(url)
		if status != "completed" {
			break
		}
	}
}
func doGitOperationsForWorkflowFile(fileName string) {
	err1 := AddFile(fileName)
	if err1 != nil {
		log.Fatalln(err1)
	}
	err2 := CommitFile(fileName)
	if err2 != nil {
		log.Fatalln(err2)
	}
	err3 := pushCode()
	if err3 != nil {
		log.Fatalln(err3)
	}
}

// Get user/repo name of current repo
func getNames() string {
	cmd, err := GitCommand("config", "--get", "remote.origin.url")
	if err != nil {
		fmt.Println("Error while getting repo config", err)
	}
	out, errO := PrepareCmd(cmd).Output()
	if errO != nil {
		log.Fatalln(errO)
	}
	var origin = string(out)
	newOri := strings.Trim(origin, ".git\n")
	u, epr := url.Parse(newOri)
	if epr != nil {
		log.Fatal(epr)
	}
	return u.Path
}

// Method to push the current local branch to remote
func pushTheBranch(name string) error {
	branch := strings.ReplaceAll(name, ".yml", "")
	fmt.Println("Pushing the branch to remote..")
	pushCmd, err := GitCommand("push", "--set-upstream", "origin", branch)
	if err != nil {
		return err
	}
	return PrepareCmd(pushCmd).Run()
}

func pushCode() error {
	pushCmd, err := GitCommand("push")
	if err != nil {
		return err
	}
	return PrepareCmd(pushCmd).Run()
}

// CheckoutBranch Checkout Branch from master
func CheckoutBranch(filename string) error {
	branch := strings.ReplaceAll(filename, ".yml", "")
	configCmd, err := GitCommand("checkout", "-b", branch)
	if err != nil {
		return err
	}
	return PrepareCmd(configCmd).Run()
}
func MoveFile(fileName string) {
	filePath := fmt.Sprintf(".github/workflows/%s", fileName)
	cmd, err := exec.Command("mv", fileName, filePath).Output()
	if err != nil {
		fmt.Println("Error while creating a PR", err)
	}
	fmt.Println(string(cmd))
}

func AddFile(fileName string) error {
	filePath := fmt.Sprintf(".github/workflows/%s", fileName)
	configCmd, err := GitCommand("add", filePath)
	if err != nil {
		return err
	}
	return PrepareCmd(configCmd).Run()
}

func CommitFile(fileName string) error {
	configCmd, err := GitCommand("commit", "-m", fileName)
	if err != nil {
		return err
	}
	return PrepareCmd(configCmd).Run()
}

func RunWorkflow(fileName string) error {
	branch := strings.ReplaceAll(fileName, ".yml", "")
	cmd, err := exec.Command("gh", "workflow", "run", fileName, "--ref", branch).Output()
	if err != nil {
		fmt.Println("Error while triggering the workflow run", err)
	}
	fmt.Println(string(cmd))
	return err
}

// Fetch all the release tags available for stack repository
func getReleases(url string) ([]Release, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	var release []Release
	parseError := json.NewDecoder(resp.Body).Decode(&release)
	defer resp.Body.Close()
	return release, parseError

}

// GitCommand Misc Functions
func GitCommand(args ...string) (*exec.Cmd, error) {
	gitExe, err := safeexec.LookPath("git")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			programName := "git"
			if runtime.GOOS == "windows" {
				programName = "Git for Windows"
			}
			return nil, &NotInstalled{
				message: fmt.Sprintf("unable to find git executable in PATH; please install %s before retrying", programName),
				error:   err,
			}
		}
		return nil, err
	}
	return exec.Command(gitExe, args...), nil
}

type NotInstalled struct {
	message string
	error
}

func (e *NotInstalled) Error() string {
	return e.message
}

// Runnable is typically an exec.Cmd or its stub in tests
type Runnable interface {
	Output() ([]byte, error)
	Run() error
}

// PrepareCmd extends exec.Cmd with extra error reporting features and provides a
// hook to stub command execution in tests
var PrepareCmd = func(cmd *exec.Cmd) Runnable {
	return &cmdWithStderr{cmd}
}

// cmdWithStderr augments exec.Cmd by adding stderr to the error message
type cmdWithStderr struct {
	*exec.Cmd
}

func (c cmdWithStderr) Output() ([]byte, error) {
	if os.Getenv("DEBUG") != "" {
		_ = printArgs(os.Stderr, c.Cmd.Args)
	}
	if c.Cmd.Stderr != nil {
		return c.Cmd.Output()
	}
	errStream := &bytes.Buffer{}
	c.Cmd.Stderr = errStream
	out, err := c.Cmd.Output()
	if err != nil {
		err = &CmdError{errStream, c.Cmd.Args, err}
	}
	return out, err
}

func (c cmdWithStderr) Run() error {
	if os.Getenv("DEBUG") != "" {
		_ = printArgs(os.Stderr, c.Cmd.Args)
	}
	if c.Cmd.Stderr != nil {
		return c.Cmd.Run()
	}
	errStream := &bytes.Buffer{}
	c.Cmd.Stderr = errStream
	err := c.Cmd.Run()
	if err != nil {
		err = &CmdError{errStream, c.Cmd.Args, err}
	}
	return err
}

// CmdError provides more visibility into why an exec.Cmd had failed
type CmdError struct {
	Stderr *bytes.Buffer
	Args   []string
	Err    error
}

func (e CmdError) Error() string {
	msg := e.Stderr.String()
	if msg != "" && !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	return fmt.Sprintf("%s%s: %s", msg, e.Args[0], e.Err)
}

func printArgs(w io.Writer, args []string) error {
	if len(args) > 0 {
		// print commands, but omit the full path to an executable
		args = append([]string{filepath.Base(args[0])}, args[1:]...)
	}
	_, err := fmt.Fprintf(w, "%v\n", args)
	return err
}

// Method to download and save patch file
func downloadTheWorkflowFile(filename string, fileUrl string) bool {
	fmt.Println("Downloading Workflow files...")
	var fileLen = 0
	out, _ := os.Create(filename)
	// timeout if it takes more than 10 secs
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fileUrl)
	if err != nil {
		log.Fatalln("Timeout", err.Error())
	}
	_, _ = io.Copy(out, resp.Body)
	fileLen++
	resp.Body.Close()
	out.Close()

	fmt.Println("Download complete for workflow files.")
	return fileLen == 1
}

// Invoke workflow run
func getWorkflowRunStats(fileUrl string) (Workflows, error) {
	resp, err := http.Get(fileUrl)
	if err != nil {
		log.Fatalln(err)
	}
	var workflow Workflows
	parseError := json.NewDecoder(resp.Body).Decode(&workflow)
	defer resp.Body.Close()
	return workflow, parseError
}
