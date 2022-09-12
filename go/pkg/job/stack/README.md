# Introduction

This package contains functionality for a FILO buffer

## Interface

To be a stacker this interface must be followed

```
type Stacker interface {
	Push(v interface{})
	Pop() (interface{}, bool)
	Len() int
}
```

## Stack

Simple filo stack

### Usage

```
s := stack.New()
s.Push("test1")
s.Push("test2")
s.Push("test3")

// Traverse stack
for v := range stack.Iter(s) {
    log.Println(v)
}
// prints:
//   test3
//   test2
//   test1
```

## Synced stack

This is a thread safe version of any type implementing a stacker

### Usage
```
s := stack.NewSyncedStack(stack.New(), /* block*/ true)
s.Push("test1")
s.Push("test2")
s.Push("test3")

// Traverse stack
go func() {
    for v := range stack.Iter(s) {
        log.Println(v)
    }
    // Blocks until stack is unblocked.
}()

s.Unblock() // Blocks until the stack is empty, and then makes the stack non-blocking.

// prints:
//   test3
//   test2
//   test1

```

## Keyed stack

This is a keyed version of any type implementing a stacker

### Interface

Elements pushed onto a keyed stack must implement the Identifiable interface.

```
type Identifiable interface {
	ID() string
}
```

NOTE: The type checking of this is done runtime!


### Usage

```
type mydata struct {
    ID string
    Name string
}

func (d *mydata) ID string {
    return d.ID
}

s := stack.NewKeyedStack(stack.New())
s.Push(&mydata{ID: "id1", Name: "name1"})
s.Push(&mydata{ID: "id2", Name: "name2"})
s.Push(&mydata{ID: "id3", Name: "name3"})
s.Push(&mydata{ID: "id1", Name: "name4"})

// Traverse stack
for v := range stack.Iter(s) {
    log.Printf("%#v", v)
}

// prints:
//   &mydata{ID: "id1", Name: "name4"}
//   &mydata{ID: "id3", Name: "name3"}
//   &mydata{ID: "id2", Name: "name2"}
```
