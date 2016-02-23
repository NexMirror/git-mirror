package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
)

var target *regexp.Regexp

func init() {
	target = regexp.MustCompile("(?m)^target$")
}

func mirror(cfg config, r repo) error {
	var cmd *exec.Cmd
	repoPath := path.Join(cfg.BasePath, r.Name)

	if checkRemoteTarget(repoPath) {
		cmd = exec.Command("git", "remote", "remove", "target")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to remove remote %s, %s", r.Target, err)
		}
	}

	if _, err := os.Stat(repoPath); err == nil {
		// Directory exists, update.
		cmd = exec.Command("git", "remote", "update")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update remote in %s, %s", repoPath, err)
		}
	} else if os.IsNotExist(err) {
		// Clone
		parent := path.Dir(repoPath)
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for cloning %s, %s", repoPath, err)
		}
		cmd = exec.Command("git", "clone", "--mirror", r.Origin, repoPath)
		cmd.Dir = parent
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone %s, %s", r.Origin, err)
		}
	} else {
		return fmt.Errorf("failed to stat %s, %s", repoPath, err)
	}

	cmd = exec.Command("git", "update-server-info")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update-server-info for %s, %s", repoPath, err)
	}

	if r.Target != "" {
		if !checkRemoteTarget(repoPath) {
			cmd = exec.Command("git", "remote", "add", "--mirror=push", "target", r.Target)
			cmd.Dir = repoPath
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to add remote %s, %s", r.Target, err)
			}
		}

		gitSshPath, err := filepath.Abs("git-ssh")
		if err != nil {
			return fmt.Errorf("unable to get absolute path to git-ssh, %s", err)
		}
		cmd = exec.Command("git", "push", "target")
		cmd.Dir = repoPath
		env := os.Environ()
		env = append(env, "GIT_SSH=" + gitSshPath)
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to push to %s, %s", r.Target, err)
		}
	}

	return nil
}

func checkRemoteTarget(repoPath string) bool {
	cmd := exec.Command("git", "remote")
	cmd.Dir = repoPath
	output, _ := cmd.Output()
	return string(target.Find(output)) == "target"
}
