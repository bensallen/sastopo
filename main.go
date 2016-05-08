package main

import (
	"fmt"

	"gitlab.alcf.anl.gov/jlse/sastopo/lib"
)

func main() {
	devices, err := sastopo.ScsiDevices()
	if err != nil {
		fmt.Print(err)
	}
	for _, device := range devices {
		fmt.Printf("Found Device: %#v \n", device.OtherPaths)
	}

}
