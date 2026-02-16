package main

import (
	"flag"

	"github.com/julianstephens/kjv-sources/tools/util"
)

func main() {

	subcommand := flag.String("cmd", "", "Subcommand to run (e.g. 'books', 'aliases')")
	flag.Parse()

	stop := make(chan bool)

	switch *subcommand {
	case "books":
		go util.Spinner("Extracting books", stop)
		MainBooks(stop)
	case "aliases":
		go util.Spinner("Extracting aliases", stop)
		MainAliases(stop)
	default:
		println("Please provide a valid subcommand using -cmd flag (e.g. -cmd=books or -cmd=aliases)")
	}

	if _, ok := <-stop; ok {
		close(stop)
	}
}
