package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

type RawCmd struct {
	Raw string `type:"existingdir" help:"The raw HTML source directory" default:"./raw"`
}

type CanonCmd struct {
	Canon   string `type:"existingdir" help:"The output directory for processed files"      default:"./canon/kjv"`
	Indexes string `type:"existingdir" help:"The index directory containing metadata files" default:"./canon/kjv/index"`
}

type CLI struct {
	Raw   RawCmd   `cmd:"" help:"Validate raw HTML chapter files for structure and content correctness"`
	Canon CanonCmd `cmd:"" help:"Validate processed canon files for structure and content correctness"`
}

func main() {
	stop := make(chan bool)
	kongCtx := kong.Parse(
		&CLI{},
		kong.Name("kjv-verify"),
		kong.Description("KJV Verification Tool"),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Bind(stop),
	)

	if err := kongCtx.Run(); err != nil {
		if _, ok := <-stop; ok {
			close(stop)
		}
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if _, ok := <-stop; ok {
		close(stop)
	}
}
