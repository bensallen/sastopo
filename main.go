package main

import (
	"fmt"

	"gitlab.alcf.anl.gov/jlse/sastopo/lib"
)

func main() {
	devices, err := scsi.SgDevices("/proc/scsi/sg/devices")
	if err != nil {
		fmt.Print(err)
	}
	for _, device := range devices {
		fmt.Print(*device)
	}

}
