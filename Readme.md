![Build Status](https://github.com/fieldryand/goflow/actions/workflows/go.yml/badge.svg)
[![codecov](https://codecov.io/gh/fieldryand/goflow/branch/master/graph/badge.svg)](https://codecov.io/gh/fieldryand/goflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/fieldryand/goflow)](https://goreportcard.com/report/github.com/fieldryand/goflow)
[![GoDoc](https://pkg.go.dev/badge/github.com/fieldryand/goflow?status.svg)](https://pkg.go.dev/github.com/fieldryand/goflow?tab=doc)
[![Release](https://img.shields.io/github/v/release/fieldryand/goflow)](https://github.com/fieldryand/goflow/releases)

# Goflow

A simple but powerful DAG scheduler and dashboard, written in Go.

> Note: this document describes the [v2 release](https://github.com/fieldryand/goflow/releases), which is still in beta.

## Contents

- [Quick start](#quick-start)
   - [With Docker](#with-docker)
   - [Without Docker](#without-docker)
- [Why Goflow?](#why-goflow)
- [Development overview](#development-overview)
   - [Jobs and tasks](#jobs-and-tasks)
   - [Custom Operators](#custom-operators)
   - [Retries](#retries)
   - [Task dependencies](#task-dependencies)
   - [Trigger rules](#trigger-rules)
   - [The Goflow engine](#the-goflow-engine)
   - [Available operators](#available-operators)
- [API and integration](#api-and-integration)

## Quick start

### With Docker

```shell
docker run -p 8181:8181 ghcr.io/fieldryand/goflow-example:latest
```

Check out the dashboard at `localhost:8181`.

![goflow-demo](https://user-images.githubusercontent.com/3333324/147818084-ade84547-4404-4d58-a697-c18ecb06fd30.gif)

### Without Docker

In a fresh project directory:

```shell
go mod init # create a new module
go get github.com/fieldryand/goflow/v2 # install dependencies
```

Create a file `main.go` with contents:
```go
package main

import "github.com/fieldryand/goflow/v2"

func main() {
        options := goflow.Options{
                UIPath: "ui/",
                StreamJobRuns: true,
                ShowExamples:  true,
        }
        gf := goflow.New(options)
        gf.Use(goflow.DefaultLogger())
        gf.Run(":8181")
}
```

Download and untar the dashboard:

```shell
wget https://github.com/fieldryand/goflow/releases/latest/download/goflow-ui.tar.gz
tar -xvzf goflow-ui.tar.gz
rm goflow-ui.tar.gz
```

Now run the application with `go run main.go` and see it in the browser at localhost:8181.

## Why Goflow?

Goflow was built as a simple replacement for Apache Airflow, which started to feel too heavy for projects where all the computation was offloaded to independent services. Still there was a need for scheduling, orchestration, concurrency, retries, a dashboard, etc. Compared to other DAG schedulers, Goflow lets you deliver all this in a single binary, which is easily deployed and runs comfortably on a single tiny VM. That's why Goflow is great for minimizing cloud costs. However, this does mean fewer capabilities in terms of scalability and throughput. There is currently no support for job queueing and distributed workers, so if you need those features, you should prefer one of the many other solutions like Airflow or Temporal.

Also, in comparison to other DAG schedulers, Goflow assumes you prefer to define your DAGs with code rather than configuration files. This approach can have various advantages, including easier testing.

## Development overview

First a few definitions.

- `Job`: A Goflow workflow is called a `Job`. Jobs can be scheduled using cron syntax.
- `Task`: Each job consists of one or more tasks organized into a dependency graph. A task can be run under certain conditions; by default, a task runs when all of its dependencies finish successfully.
- Concurrency: Jobs and tasks execute concurrently.
- `Operator`: An `Operator` defines the work done by a `Task`. Goflow comes with a handful of basic operators, and implementing your own `Operator` is straightforward.
- Retries: You can allow a `Task` a given number of retry attempts. Goflow comes with two retry strategies, `ConstantDelay` and `ExponentialBackoff`.
- Database: Goflow supports two database types, in-memory and BoltDB. BoltDB will persist your history of job runs, whereas in-memory means the history will be lost each time the Goflow server is stopped. The default is BoltDB.
- Streaming: Goflow uses server-sent events to stream the status of jobs and tasks to the dashboard in real time.

### Jobs and tasks

Let's start by creating a function that returns a job called `myJob`. There is a single task in this job that sleeps for one second.

```go
package main

import (
	"errors"

	"github.com/fieldryand/goflow/v2"
)

func myJob() *goflow.Job {
	j := &goflow.Job{Name: "myJob", Schedule: "* * * * *", Active: true}
	j.Add(&goflow.Task{
		Name:     "sleepForOneSecond",
		Operator: goflow.Command{Cmd: "sleep", Args: []string{"1"}},
	})
	return j
}
```

By setting `Active: true`, we are telling Goflow to apply the provided cron schedule for this job when the application starts.
Job scheduling can be activated and deactivated from the dashboard.

### Custom operators

A custom `Operator` needs to implement the `Run` method. Here's an example of an operator that adds two positive numbers.

```go
type PositiveAddition struct{ a, b int }

func (o PositiveAddition) Run() (interface{}, error) {
	if o.a < 0 || o.b < 0 {
		return 0, errors.New("Can't add negative numbers")
	}
	result := o.a + o.b
	return result, nil
}
```

### Retries

Let's add a retry strategy to the `sleepForOneSecond` task:

```go
func myJob() *goflow.Job {
	j := &goflow.Job{Name: "myJob", Schedule: "* * * * *"}
	j.Add(&goflow.Task{
		Name:       "sleepForOneSecond",
		Operator:   goflow.Command{Cmd: "sleep", Args: []string{"1"}},
		Retries:    5,
		RetryDelay: goflow.ConstantDelay{Period: 1},
	})
	return j
}
```

Instead of `ConstantDelay`, we could also use `ExponentialBackoff` (see https://en.wikipedia.org/wiki/Exponential_backoff).

### Task dependencies

A job can define a directed acyclic graph (DAG) of independent and dependent tasks. Let's use the `SetDownstream` method to
define two tasks that are dependent on `sleepForOneSecond`. The tasks will use the `PositiveAddition` operator we defined earlier,
as well as a new operator provided by Goflow, `Get`.

```go
func myJob() *goflow.Job {
	j := &goflow.Job{Name: "myJob", Schedule: "* * * * *"}
	j.Add(&goflow.Task{
		Name:       "sleepForOneSecond",
		Operator:   goflow.Command{Cmd: "sleep", Args: []string{"1"}},
		Retries:    5,
		RetryDelay: goflow.ConstantDelay{Period: 1},
	})
	j.Add(&goflow.Task{
		Name:       "getGoogle",
		Operator:   goflow.Get{Client: &http.Client{}, URL: "https://www.google.com"},
	})
	j.Add(&goflow.Task{
		Name:       "AddTwoPlusThree",
		Operator:   PositiveAddition{a: 2, b: 3},
	})
	j.SetDownstream(j.Task("sleepForOneSecond"), j.Task("getGoogle"))
	j.SetDownstream(j.Task("sleepForOneSecond"), j.Task("AddTwoPlusThree"))
	return j
}
```

### Trigger rules

By default, a task has the trigger rule `allSuccessful`, meaning the task starts executing when all the tasks directly
upstream exit successfully. If any dependency exits with an error, all downstream tasks are skipped, and the job exits with an error.

Sometimes you want a downstream task to execute even if there are upstream failures. Often these are situations where you want
to perform some cleanup task, such as shutting down a server. In such cases, you can give a task the trigger rule `allDone`.

Let's modify `sleepForOneSecond` to have the trigger rule `allDone`.


```go
func myJob() *goflow.Job {
	// other stuff
	j.Add(&goflow.Task{
		Name:        "sleepForOneSecond",
		Operator:    goflow.Command{Cmd: "sleep", Args: []string{"1"}},
		Retries:     5,
		RetryDelay:  goflow.ConstantDelay{Period: 1},
		TriggerRule: "allDone",
	})
	// other stuff
}
```

### The Goflow Engine

Finally, let's create a Goflow engine, register our job, attach a logger, and run the application.

```go
func main() {
	gf := goflow.New(goflow.Options{StreamJobRuns: true})
	gf.AddJob(myJob)
	gf.Use(goflow.DefaultLogger())
	gf.Run(":8181")
}
```

You can pass different options to the engine. Options currently supported:
- `UIPath`: The path containing the dashboard assets. This is required to run the dashboard. Recommended value: `ui/`
- `DBType`: `boltdb` (default) or `memory`.
- `BoltDBPath`: This will be the filepath of the Bolt database on disk. Default value: `goflow.db`
- `StreamJobRuns`: Whether to stream updates to the dashboard. Recommended value: `true`
- `ShowExamples`: Whether to show the example jobs. Default value: `false`

Goflow is built on the [Gin framework](https://github.com/gin-gonic/gin), so you can pass any Gin handler to `Use`.

### Available operators

Goflow provides several operators for common tasks. [See the package documentation](https://pkg.go.dev/github.com/fieldryand/goflow) for details on each.

- `Command` executes a shell command.
- `Get` makes a GET request.
- `Post` makes a POST request.

## API and integration

You can use the API to integrate Goflow with other applications, such as an existing dashboard. Here is an overview of available endpoints:
- `GET /api/health`: Check health of the service
- `GET /api/jobs`: List registered jobs
- `GET /api/jobs/{jobname}`: Get the details for a given job
- `GET /api/jobruns`: Query and list jobruns
- `POST /api/jobs/{jobname}/submit`: Submit a job for execution
- `POST /api/jobs/{jobname}/toggle`: Toggle a job schedule on or off
- `/stream`: This endpoint returns Server-Sent Events with a `data` payload matching the one returned by `/api/jobruns`. The dashboard that ships with Goflow uses this endpoint.

Check out the OpenAPI spec for more details. Easiest way is to clone the repo, then within the repo use Swagger as in the following:

```shell
docker run -p 8080:8080 -e SWAGGER_JSON=/app/swagger.json -v $(pwd):/app swaggerapi/swagger-ui
```
