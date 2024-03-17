// Run one or more npm-run scripts interactively

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// sh runs a shell command and returns the output as a string.
func sh(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %s, stderr: %s", err, stderr.String())
	}

	return strings.TrimSuffix(stdout.String(), "\n"), nil
}

// Receives a JSON string from 'npm run --json' and attempts to
// unmarshal it into a string:string map. Since 'npm run --json' can return
// either a root-level map or a workspace map, this function attempts to unmarshal
// both, returning the result of the first successful unmarshalling.
func unmarshalNpmRun(jsonStr string) (map[string]string, error) {
	var rootUnmarshalResult map[string]string
	var workspaceUnmarshalResult map[string]map[string]string

	rootUnmarshalErr := json.Unmarshal([]byte(jsonStr), &rootUnmarshalResult)
	if rootUnmarshalErr == nil {
		return rootUnmarshalResult, nil
	}

	workspaceUnmarshalErr := json.Unmarshal([]byte(jsonStr), &workspaceUnmarshalResult)
	if workspaceUnmarshalErr == nil {
		for _, nestedMap := range workspaceUnmarshalResult {
			return nestedMap, nil
		}
	}

	return nil, errors.New("error unmarshalling from 'npm run --json'")
}

// filter runs 'gum filter' with the given items and returns the selected item.
func filter(items []string) (string, error) {
	gumFilter := exec.Command("gum", "filter", "--height", strconv.Itoa(len(items)))
	in := bytes.NewBufferString(strings.Join(items, "\n"))
	gumFilter.Stdin = in
	gumFilter.Stderr = os.Stderr

	var out bytes.Buffer
	gumFilter.Stdout = &out

	err := gumFilter.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(out.String(), "\n"), nil
}

// shStreamOutput runs a shell command and streams the output to stdout.
func shStreamOutput(bin string, args ...string) error {
	fmt.Println(">", bin, strings.Join(args, " "))
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	jsonStr, err := sh("npm", "run", "--json")
	if err != nil {
		log.Fatalln("couldn't read scripts from package.json")
	}
	jsonRun, err := unmarshalNpmRun(jsonStr)
	if err != nil {
		log.Fatalln("couldn't unmarshal npm run --json output")
	}

	items := make([]string, 0, len(jsonRun))
	for k := range jsonRun {
		items = append(items, k)
	}
	item, err := filter(items)
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			userHitCtrlC := exitError.ExitCode() == 130
			if userHitCtrlC {
				os.Exit(0)
			}
		} else {
			log.Fatalln("error running gum filter")
		}
	}

	noValidItemSelected := item == ""
	if noValidItemSelected {
		os.Exit(0)
	}

	err = shStreamOutput("npm", "run", item)
	if err != nil {
		log.Println("error running npm script:", item)
	}
}
