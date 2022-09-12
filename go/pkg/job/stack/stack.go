package stack

type Stacker interface {
	Push(v interface{})
	Pop() (interface{}, bool)
	Len() int
}

type Stack struct {
	top    *node
	length int
}

type node struct {
	value interface{}
	prev  *node
}

func New() *Stack {
	return &Stack{}
}

// Return the number of items in the stack
func (this *Stack) Len() int {
	return this.length
}

// Pop the top item of the stack and return it
func (this *Stack) Pop() (interface{}, bool) {
	if this.length == 0 {
		return nil, false
	}

	n := this.top
	this.top = n.prev
	this.length--
	return n.value, true
}

// Push a value onto the top of the stack
func (this *Stack) Push(value interface{}) {
	n := &node{value, this.top}
	this.top = n
	this.length++
}

func Iter(stack Stacker) chan interface{} {
	values := make(chan interface{})
	go func() {
		for {
			v, ok := stack.Pop()
			if !ok {
				break
			}
			values <- v
		}
		close(values)
	}()
	return values
}
