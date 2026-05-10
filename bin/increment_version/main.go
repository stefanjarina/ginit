package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type version struct {
	tag   string
	major int
	minor int
	patch int
}

var semverTag = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

func main() {
	major := flag.Bool("major", false, "Increment major version")
	minor := flag.Bool("minor", false, "Increment minor version")
	dryRun := flag.Bool("dry-run", false, "Print the tag that would be created without creating it")
	yes := flag.Bool("yes", false, "Create tag without prompting")
	shortYes := flag.Bool("y", false, "Create tag without prompting")
	help := flag.Bool("help", false, "Show help")
	shortHelp := flag.Bool("h", false, "Show help")

	flag.Usage = func() {
		fmt.Println("Usage: increment_version.go [--major | --minor] [--dry-run] [--yes]")
	}

	flag.Parse()

	if *help || *shortHelp {
		flag.Usage()
		return
	}

	part := "patch"
	if *major {
		part = "major"
	} else if *minor {
		part = "minor"
	}

	current, err := currentVersion()
	if err != nil {
		fmt.Printf("Error getting current version: %v\n", err)
		os.Exit(1)
	}

	newVersion := nextVersion(current, part)

	fmt.Printf("Current version: %s\n", current.tag)
	fmt.Printf("New version:     %s (%s)\n", newVersion, part)

	if *dryRun {
		fmt.Printf("Dry run: would create tag %s\n", newVersion)
		return
	}

	if !(*yes || *shortYes) && !confirmTag() {
		fmt.Println("Aborted.")
		return
	}

	if err := runGit("tag", newVersion); err != nil {
		fmt.Printf("Error creating tag: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Created tag %s\n", newVersion)
}

func currentVersion() (version, error) {
	output, err := gitOutput("tag", "--list")
	if err != nil {
		return version{}, err
	}

	versions := parseVersions(output)
	if len(versions) == 0 {
		return version{tag: "v0.0.0"}, nil
	}

	sort.Slice(versions, func(i, j int) bool {
		if versions[i].major != versions[j].major {
			return versions[i].major < versions[j].major
		}
		if versions[i].minor != versions[j].minor {
			return versions[i].minor < versions[j].minor
		}
		return versions[i].patch < versions[j].patch
	})

	return versions[len(versions)-1], nil
}

func parseVersions(tags string) []version {
	lines := strings.Split(strings.TrimSpace(tags), "\n")
	versions := make([]version, 0, len(lines))

	for _, line := range lines {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}

		matches := semverTag.FindStringSubmatch(tag)
		if matches == nil {
			continue
		}

		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])
		patch, _ := strconv.Atoi(matches[3])

		versions = append(versions, version{
			tag:   tag,
			major: major,
			minor: minor,
			patch: patch,
		})
	}

	return versions
}

func nextVersion(current version, part string) string {
	switch part {
	case "major":
		return fmt.Sprintf("v%d.0.0", current.major+1)
	case "minor":
		return fmt.Sprintf("v%d.%d.0", current.major, current.minor+1)
	default:
		return fmt.Sprintf("v%d.%d.%d", current.major, current.minor, current.patch+1)
	}
}

func confirmTag() bool {
	fmt.Print("\nCreate this tag? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil && len(response) == 0 {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", formatCommandError(err, stderr.String())
	}

	return string(output), nil
}

func runGit(args ...string) error {
	cmd := exec.Command("git", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return formatCommandError(err, stderr.String())
	}

	return nil
}

func formatCommandError(err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}

	return fmt.Errorf("%w: %s", err, stderr)
}
