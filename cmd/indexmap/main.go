package main

import (
	"flag"
	"fmt"
	"os"
	
	"github.com/suparena/entitystore"
	"github.com/suparena/entitystore/processor"
)

var (
	versionFlag = flag.Bool("version", false, "Show version information")
	vFlag       = flag.Bool("v", false, "Show version information (short)")
)

func main() {
	// Parse flags early to catch version flag
	flag.Parse()
	
	// Handle version flag
	if *versionFlag || *vFlag {
		info := entitystore.GetVersionInfo()
		fmt.Printf("EntityStore indexmap-pps version %s\n", info.Version)
		fmt.Printf("Git commit: %s\n", info.GitCommit)
		fmt.Printf("Build date: %s\n", info.BuildDate)
		fmt.Printf("Go version: %s\n", info.GoVersion)
		os.Exit(0)
	}
	
	// Run the processor
	processor.Main()
}