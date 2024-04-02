package main

import (
	"fmt"
	"os"

	nz "github.com/dextryz/notezero"
)

func main() {
	err := nz.Main()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
