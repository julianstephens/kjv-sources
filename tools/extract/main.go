package main

import (
	"flag"
)

func main() {

	subcommand := flag.String("cmd", "", "Subcommand to run (e.g. 'books', 'aliases')")
	flag.Parse()

	switch *subcommand {
	case "books":
		MainBooks()
	case "aliases":
		MainAliases()
	default:
		println("Please provide a valid subcommand using -cmd flag (e.g. -cmd=books or -cmd=aliases)")
	}

}
