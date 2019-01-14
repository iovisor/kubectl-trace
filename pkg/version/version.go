package version

import (
	"fmt"
	"strconv"
	"time"
)

// Populated by makefile
var gitCommit string
var imageName string
var buildTime string
var versionFormat = "git commit: %s\nbuild date: %s"
var imageNameTagFormat = "%s:%s"

// ImageName returns the container image name defined in Makefile
func ImageName() string {
	return imageName
}

func GitCommit() string {
	return gitCommit
}

func ImageNameTag() string {
	commit := GitCommit()
	if len(commit) == 0 {
		return fmt.Sprintf(imageNameTagFormat, ImageName(), "latest")
	}
	return fmt.Sprintf(imageNameTagFormat, ImageName(), GitCommit())
}

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

func String() string {
	ts := Time()
	if ts == nil {
		return fmt.Sprintf(versionFormat, GitCommit(), "undefined")
	}
	return fmt.Sprintf(versionFormat, GitCommit(), ts.String())
}
