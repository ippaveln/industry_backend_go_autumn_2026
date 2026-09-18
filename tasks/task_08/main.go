package main

import (
	"fmt"
	"time"
)

type demoClock struct{}

func (demoClock) Now() time.Time { return time.Unix(0, 0) }
func main()                      { l := NewLimiter(demoClock{}, 0, 3); fmt.Println(l.AllowN(2), l.AllowN(2), l.AllowN(1)) }
