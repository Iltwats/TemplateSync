package main

import (
	"fmt"
	"os/exec"
)

// Method to raise a pull request
func raiseAPullRequest() {
	fmt.Println("Creating a pull request...")
	prTitle := "Migration-patch"
	prBody := "PR to migrate to latest stack version"
	cmd, err := exec.Command("gh", "pr", "create", "--title", prTitle, "--body", prBody).Output()
	if err != nil {
		fmt.Println("Error while creating a PR", err)
	}
	fmt.Println("To complete the merge, merge this PR by going to the following link: ", string(cmd))
}
