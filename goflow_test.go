package goflow

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fieldryand/goflow/operator"
)

func TestJobsRoute(t *testing.T) {
	jobs := map[string](func() *Job){"example": ExampleJob}
	router := Goflow(jobs)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/jobs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/jobs status is %d, expected %d", w.Code, http.StatusOK)
	}
}

func ExampleJob() *Job {
	sleep_1 := NewTask("sleep 1", operator.NewSleep(1))
	j := NewJob("example").AddTask(sleep_1)
	return j
}