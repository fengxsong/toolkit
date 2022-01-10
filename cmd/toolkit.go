package main

import (
	"fmt"
	"os"

	"github.com/fengxsong/toolkit/cmd/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
