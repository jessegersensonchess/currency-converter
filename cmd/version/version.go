// main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func incrementVersion(version string, part string) (string, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version format: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid major version: %v", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid minor version: %v", err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid patch version: %v", err)
	}

	switch part {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	default:
		return "", fmt.Errorf("unknown part to increment: %s", part)
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

func determinePart(title string) string {
	// Define regex patterns for [major] and [minor], case-insensitive
	majorPattern := regexp.MustCompile(`(?i)\[major\]`)
	minorPattern := regexp.MustCompile(`(?i)\[minor\]`)

	if majorPattern.MatchString(title) {
		return "major"
	} else if minorPattern.MatchString(title) {
		return "minor"
	}
	return "patch"
}

func main() {
	versionPtr := flag.String("version", "0.0.0", "Current version in semantic versioning format (e.g., 1.2.3)")
	titlePtr := flag.String("title", "", "Pull Request title containing [major] or [minor] to indicate version increment")
	flag.Parse()

	if *titlePtr == "" {
		fmt.Fprintln(os.Stderr, "Error: -title flag is required")
		os.Exit(1)
	}

	part := determinePart(*titlePtr)

	newVersion, err := incrementVersion(*versionPtr, part)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error incrementing version: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s", newVersion)

	// Write the new version to GITHUB_OUTPUT for GitHub Actions
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput != "" {
		f, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening GITHUB_OUTPUT: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		_, err = fmt.Fprintf(f, "new_version=%s\n", newVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to GITHUB_OUTPUT: %v\n", err)
			os.Exit(1)
		}
	}
}
