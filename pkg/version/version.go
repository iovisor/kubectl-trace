package version

import (
	"fmt"
	"strconv"
	"time"
)

// Populated by makefile
var gitCommit string
var buildTime string
var versionFormat = "git commit: %s\nbuild date: %s"

// GitCommit returns the git commit
func GitCommit() string {
	return gitCommit
}

// Time returns the build time
func Time() *time.Time {
	if len(buildTime) == 0 {
		return nil
	}
	i, err := strconv.ParseInt(buildTime, 10, 64)
	if err != nil {
		return nil
	}
	t := time.Unix(i, 0)
	return &t
}

// String returns version info as a string
func String() string {
	ts := Time()
	if ts == nil {
		return fmt.Sprintf(versionFormat, GitCommit(), "undefined")
	}
	return fmt.Sprintf(versionFormat, GitCommit(), ts.String())
}
