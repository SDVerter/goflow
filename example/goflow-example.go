package main

import "github.com/fieldryand/goflow"

func main() {
	options := goflow.Options{
		AssetBasePath: "ui/",
		StreamJobRuns: true,
		ShowExamples:  true,
	}
	gf := goflow.New(options)
	gf.Use(goflow.DefaultLogger())
	gf.Run(":8181")
}
