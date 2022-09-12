package job

import (
	"nathejk.dk/pkg/job/stack"
)

type Runner struct {
	jobs       *stack.SyncedStack
	keyedStack *stack.KeyedStack
}

type Interface interface {
	ID() string
	Run()
}

func NewRunner() *Runner {
	keyedStack := stack.NewKeyedStack(stack.New())
	runner := &Runner{
		jobs:       stack.NewSyncedStack(keyedStack, true),
		keyedStack: keyedStack,
	}
	return runner
}

func (j *Runner) Add(id string, callback func()) {
	j.jobs.Push(&Job{
		id:       id,
		callback: callback,
	})
}

func (j *Runner) Done() {
	j.jobs.Unblock()
}

func (j *Runner) Run() {
	j.jobs.Block()
	for job := range stack.Iter(j.jobs) {
		//log.Printf("Got job %#v", job)
		job.(Interface).Run()
	}
}

func (j *Runner) Skipped() int {
	return j.keyedStack.SkipCount()
}

func (j *Runner) Remaining() int {
	return j.jobs.Len()
}
