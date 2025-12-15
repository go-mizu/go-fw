package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type framework struct {
	Title string
	Dir   string
}

var frameworks = []framework{
	{"net/http", "nethttp"},
	{"Chi", "chi"},
	{"Gin", "gin"},
	{"Echo", "echo"},
	{"Fiber", "fiber"},
	{"Mizu", "mizu"},
}

type result struct {
	Written []string
	Missing []string
}

func main() {
	chapters, err := filepath.Glob("[0-9][0-9]-*/README.md")
	if err != nil {
		panic(err)
	}

	var res result

	for _, readme := range chapters {
		processChapter(readme, &res)
	}

	printSummary(res)

	if len(res.Missing) > 0 {
		os.Exit(1)
	}
}

func processChapter(readme string, res *result) {
	data, err := os.ReadFile(readme)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(data), "\n")
	chapterDir := filepath.Dir(readme)

	for _, fw := range frameworks {
		code := extractCode(lines, fw.Title)
		target := filepath.Join(chapterDir, fw.Dir, "main.go")

		if code == "" {
			res.Missing = append(res.Missing, target)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			panic(err)
		}

		if err := os.WriteFile(target, []byte(code), 0644); err != nil {
			panic(err)
		}

		res.Written = append(res.Written, target)
	}
}

func extractCode(lines []string, title string) string {
	inSection := false
	inCode := false
	var out []string

	sectionHeader := "## " + title

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			inSection = line == sectionHeader
			inCode = false
			continue
		}

		if !inSection {
			continue
		}

		if strings.HasPrefix(line, "```go") {
			inCode = true
			continue
		}

		if strings.HasPrefix(line, "```") && inCode {
			break
		}

		if inCode {
			out = append(out, line)
		}
	}

	if len(out) == 0 {
		return ""
	}

	return strings.Join(out, "\n") + "\n"
}

func printSummary(res result) {
	fmt.Println("summary")
	fmt.Println("-------")

	fmt.Printf("written: %d\n", len(res.Written))
	fmt.Printf("missing: %d\n", len(res.Missing))

	if len(res.Missing) == 0 {
		fmt.Println("\nall main.go files generated successfully")
		return
	}

	fmt.Println("\nmissing main.go files:")
	for _, m := range res.Missing {
		fmt.Println("-", m)
	}
}
