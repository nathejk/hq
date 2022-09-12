package job_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"nathejk.dk/pkg/job"
)

func testinc(inc *int64) func() {
	return func() {
		atomic.AddInt64(inc, 1)
	}
}

func TestJob(t *testing.T) {
	runner := job.NewRunner()

	var inc int64
	runner.Add("job1", testinc(&inc))
	runner.Add("job2", testinc(&inc))
	runner.Add("job3", testinc(&inc))
	runner.Add("job2", testinc(&inc))
	runner.Add("job3", testinc(&inc))
	runner.Add("job4", testinc(&inc))

	assert.Equal(t, 0, runner.Skipped())
	assert.Equal(t, 6, runner.Remaining())

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		runner.Run()
		wg.Done()
	}()
	time.Sleep(100 * time.Millisecond)
	runner.Done()
	wg.Wait()

	assert.Equal(t, int64(4), inc)
	assert.Equal(t, 2, runner.Skipped())
	assert.Equal(t, 0, runner.Remaining())
}
