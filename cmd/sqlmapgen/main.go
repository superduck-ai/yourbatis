package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/superduck-ai/yourbatis/internal/generator"
)

func main() {
	var config generator.Config
	flag.StringVar(&config.Mapper, "mapper", "", "Go mapper interface name")
	flag.StringVar(&config.SQLFile, "sql", "", "MyBatis mapper XML file")
	flag.StringVar(&config.Output, "out", "", "generated Go output file")
	flag.StringVar(&config.PackageDir, "dir", ".", "Go package directory")
	flag.StringVar(&config.RuntimeImport, "runtime-import", "github.com/superduck-ai/yourbatis", "yourbatis runtime import path")
	flag.StringVar(&config.Dialect, "dialect", "", "target database dialect: postgres, sqlite, or mysql")
	flag.Parse()

	if config.Mapper == "" || config.SQLFile == "" {
		fmt.Fprintln(os.Stderr, "sqlmapgen: -mapper and -sql are required")
		flag.Usage()
		os.Exit(2)
	}
	if err := generator.Generate(config); err != nil {
		fmt.Fprintf(os.Stderr, "sqlmapgen: %v\n", err)
		os.Exit(1)
	}
}
