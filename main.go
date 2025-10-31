package main

import "github.com/loickal/newsletter-cli/cmd"

var version = "0.1.3"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}

func Version() string {
	return version
}
