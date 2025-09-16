package websocket

import "fmt"

type Stack struct {
    data []int
    size int
}

func NewStack(cap int) *Stack {
    return &Stack{data: make([]int, 0, cap), size: 0}
}

func (s *Stack) Push(n int) {
    s.data = append(s.data, n)
    s.size++
}

func (s *Stack) Pop() (bool, int) {
    if s.IsEmpty() {
        return false, 0
    }
		ret := s.Top()
		
    s.size--
    s.data = s.data[:s.size]
    return true, ret
}

func (s *Stack) Top() int {
    return s.data[s.size-1]
}

func (s *Stack) IsEmpty() bool {
    return s.size == 0
}

func (s *Stack) String() string {
    return fmt.Sprint(s.data)
}