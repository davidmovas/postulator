package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	source := flag.String("source", "wp-plugin/postulator-companion", "plugin directory to package")
	out := flag.String("out", "bin/postulator-companion.zip", "zip archive to write")
	flag.Parse()

	if err := run(*source, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(*out)
}
