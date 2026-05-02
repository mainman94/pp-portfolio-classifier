package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"pp-portfolio-classifier/internal/app"
	"pp-portfolio-classifier/internal/config"
)

func main() {
	opts, err := config.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := app.Run(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
