package transformers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/joselitofilho/go-conventional-commits/internal/changelogs"
	"github.com/joselitofilho/go-conventional-commits/internal/common"
	"github.com/joselitofilho/go-conventional-commits/internal/conventionalcommits"
)

var (
	baseFormatRegex       = regexp.MustCompile(`(?is)^(?:(?P<category>[^\(!:]+)(?:\((?P<scope>[^\)]+)\))?(?P<breaking>!)?: (?P<description>[^\n\r]+))(?P<remainder>.*)`)
	bodyFooterFormatRegex = regexp.MustCompile(`(?isU)^(?:(?P<body>.*))?(?P<footer>(?-U:(?:[\w\-]+(?:: | #).*|(?i:BREAKING CHANGE:.*))+))`)
	footerFormatRegex     = regexp.MustCompile(`(?s)^(?P<footer>(?i:(?:[\w\-]+(?:: | #).*|(?i:BREAKING CHANGE:.*))+))`)
)

// TransformConventionalCommit takes a commits message and parses it into usable blocks
func TransformConventionalCommit(message string) (commit *conventionalcommits.ConventionalCommit) {
	match := baseFormatRegex.FindStringSubmatch(message)

	if len(match) == 0 {
		parts := strings.SplitN(message, "\n", 2)
		parts = append(parts, "")
		return &conventionalcommits.ConventionalCommit{
			Category:    "chore",
			Major:       strings.Contains(parts[1], "BREAKING CHANGE"),
			Description: strings.TrimSpace(parts[0]),
			Body:        strings.TrimSpace(parts[1]),
		}
	}

	result := make(map[string]string)
	regExMapper(match, baseFormatRegex, result)

	// split the remainder into body & footer
	match = bodyFooterFormatRegex.FindStringSubmatch(result["remainder"])
	if len(match) > 0 {
		regExMapper(match, bodyFooterFormatRegex, result)
	} else {
		result["body"] = result["remainder"]
	}

	for _, category := range common.MajorCategories {
		if result["category"] == category {
			result["breaking"] = "!"
			break
		}
	}

	var footers []string
	for _, v := range strings.Split(result["footer"], "\n") {
		//v = strings.TrimSpace(v)
		if !footerFormatRegex.MatchString(v) && len(footers) > 0 {
			footers[len(footers)-1] += fmt.Sprintf("\n%s", v)
			continue
		}
		footers = append(footers, v)
	}
	for i := range footers {
		footers[i] = strings.TrimSpace(footers[i])
		if footers[i] == "" { // Remove the element at index i from footers.
			copy(footers[i:], footers[i+1:])   // Shift a[i+1:] left one index.
			footers[len(footers)-1] = ""       // Erase last element (write zero value).
			footers = footers[:len(footers)-1] // Truncate slice.
		}
	}
	if len(footers) == 0 {
		footers = nil
	}

	commit = &conventionalcommits.ConventionalCommit{
		Category:    result["category"],
		Scope:       result["scope"],
		Major:       result["breaking"] == "!" || strings.Contains(result["footer"], "BREAKING CHANGE"),
		Description: result["description"],
		Body:        result["body"],
		Footer:      footers,
	}

	if commit.Major {
		return commit
	}

	for _, category := range common.MinorCategories {
		if result["category"] == category {
			commit.Minor = true
			return commit
		}
	}

	for _, category := range common.PatchCategories {
		if result["category"] == category {
			commit.Patch = true
			return commit
		}
	}

	return commit
}

func TransformConventionalCommits(messages []string) (commits conventionalcommits.ConventionalCommits) {
	for _, message := range messages {
		commits = append(commits, TransformConventionalCommit(message))
	}
	return
}

func TransformChangeLog(message string, projectLink string) *changelogs.ChangeLog {
	commit := TransformConventionalCommit(message)

	footerRefs := ""
	footerTitle := ""

	for _, footer := range commit.Footer {
		if strings.Contains(footer, "Refs #") {
			footerRefs = footer
		}
		if strings.Contains(footer, "Title: ") {
			footerTitle = footer
		}
	}

	if footerRefs != "" {
		key := strings.ReplaceAll(footerRefs, "Refs #", "")

		link := key
		if projectLink != "" {
			link = fmt.Sprintf("[%s](%s%s)", key, projectLink, key)
		}

		subject := "<put the task title here>"
		if footerTitle != "" {
			subject = strings.ReplaceAll(footerTitle, "Title: ", "")
		}

		if strings.Contains(commit.Category, "fix") {
			return &changelogs.ChangeLog{
				Category: common.Fixes,
				Refs:     key,
				Title:    subject,
				Link:     link,
			}
		} else {
			return &changelogs.ChangeLog{
				Category: common.Features,
				Refs:     key,
				Title:    subject,
				Link:     link,
			}
		}
	}

	return nil
}

func TransformChangeLogs(messages []string, projectLink string) changelogs.ChangeLogs {
	parsedChangelogs := changelogs.ChangeLogs{}

	for _, message := range messages {
		changeLog := TransformChangeLog(message, projectLink)
		if changeLog != nil {
			parsedChangelogs[changeLog.Refs] = changeLog
		}
	}

	return parsedChangelogs
}

func regExMapper(match []string, expectedFormatRegex *regexp.Regexp, result map[string]string) {
	for i, name := range expectedFormatRegex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = strings.TrimSpace(match[i])
		}
	}
}
