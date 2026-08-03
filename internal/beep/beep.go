package beep

import "fmt"

type Beep interface {
	Beep()
}

type Terminal struct{}

func (Terminal) Beep() {
	fmt.Print("\a")
}
