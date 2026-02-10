package main

import (
	"flag"
	"github.com/julianstephens/kjv-sources/tools/extract"
)

func main() {

	subcommand := flag.String("cmd", "", "Subcommand to run (e.g. 'books', 'aliases')")
	flag.Parse()

	switch *subcommand {
	case "books":
		extract.MainBooks()
	case "aliases":
		extract.MainAliases()
	default:
		println("Please provide a valid subcommand using -cmd flag (e.g. -cmd=books or -cmd=aliases)")
	}

}
