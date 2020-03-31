// +build ignore

// This program generates flow.go. It can be invoked by running
// go generate
package main

import (
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

func main() {

	files, err := ioutil.ReadDir("./jobs")
	if err != nil {
		log.Fatal(err)
	}

	var cases []string

	for _, file := range files {
		path := "./jobs/" + file.Name()
		if strings.Contains(path, "_job") {
			dat, err := ioutil.ReadFile(path)
			if err != nil {
				log.Fatal(err)
			}

			jobName, jobFn := filterFlow(string(dat))
			log.Printf("Found job %s (%s) in %s", jobName, jobFn, path)

			switchCase := "case \"" + jobName + "\": return jobs." + jobFn
			cases = append(cases, switchCase)
		}
	}

	f, _ := os.Create("./flow.go")
	defer f.Close()

	flowTemplate.Execute(f, struct{ Cases []string }{cases})
}

var flowTemplate = template.Must(template.New("").Parse(`// Code generated by go generate; DO NOT EDIT.
package main

import "github.com/fieldryand/goflow/core"
import "github.com/fieldryand/goflow/jobs"

func flow(job string) func() *core.Job {
	switch job {
	{{- range .Cases }}
		{{ . }}
	{{- end }}
	default:
		return nil
	}
}
`))

// filterFlow filters lines of source code for the flow comment
// that indicates a job. It returns the (jobName, jobFn).
func filterFlow(file string) (string, string) {
	pat := regexp.MustCompile("// goflow: .*")
	comment := strings.Split(pat.FindString(file), " ")
	return comment[3], comment[2]
}