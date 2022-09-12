package stack

type Identifiable interface {
	ID() string
}

type KeyedStack struct {
	Stack     Stacker
	ignore    map[string]bool
	skipcount int
}

func NewKeyedStack(stack Stacker) *KeyedStack {
	return &KeyedStack{
		Stack:  stack,
		ignore: make(map[string]bool),
	}
}

func (s *KeyedStack) Push(v interface{}) {
	s.Stack.Push(v)
	delete(s.ignore, v.(Identifiable).ID())
}

func (s *KeyedStack) Pop() (interface{}, bool) {
	for {
		v, ok := s.Stack.Pop()
		if !ok {
			return v, ok
		}
		id := v.(Identifiable).ID()
		if s.ignore[id] {
			s.skipcount++
			continue
		}
		s.ignore[id] = true
		return v, true
	}
}

func (s *KeyedStack) Len() int {
	return s.Stack.Len()
}

func (s *KeyedStack) SkipCount() int {
	return s.skipcount
}
