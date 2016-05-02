package main

import (
	"fmt"

	"gitlab.alcf.anl.gov/jlse/sastopo/scsi"
)

func main() {
	devices, err := scsi.SgDevices("/proc/scsi/sg/devices")
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(devices)
}
