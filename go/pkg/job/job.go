package job

type Job struct {
	id       string
	callback func()
}

func (j *Job) ID() string {
	return j.id
}

func (j *Job) Run() {
	j.callback()
}
