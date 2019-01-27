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
var imageNameTagFormat = "%s:%s"
var defaultImageName = "quay.io/fntlnz/kubectl-trace-bpftrace"
var defaultImageTag = "latest"
var defaultInitImageName = "quay.io/dalehamel/kubectl-trace-init"
var defaultInitImageTag = "latest"

// ImageName returns the container image name defined in Makefile
func ImageName() string {
	return imageName
}

// GitCommit returns the git commit
func GitCommit() string {
	return gitCommit
}

func ImageNameTag() string {
	imageName := ImageName()
	tag := GitCommit()
	if len(tag) == 0 {
		tag = defaultImageTag
	}
	if len(imageName) == 0 {
		imageName = defaultImageName
	}
	return fmt.Sprintf(imageNameTagFormat, imageName, tag)
}

func InitImageNameTag() string {
	return fmt.Sprintf(imageNameTagFormat, defaultInitImageName, defaultInitImageTag)
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
