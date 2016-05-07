package main

import (
	"fmt"

	"gitlab.alcf.anl.gov/jlse/sastopo/lib"
)

func main() {
	devices, err := sastopo.SgDevices2()
	if err != nil {
		fmt.Print(err)
	}
	for _, device := range devices {
		fmt.Printf("Found SG: %d, serial: %#v \n", device.ID, device.Serial)
	}

}
