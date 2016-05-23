package main

import (
	"fmt"

	"gitlab.alcf.anl.gov/jlse/sastopo/lib"
)

func findDevMissingPaths(count int, devices map[string]*sastopo.Device) {
	for _, d := range devices {
		if len(d.MultiPath.Paths) != count {
			fmt.Printf("Path Count Mismatch: %#v, found %d paths\n", d.Serial, len(d.MultiPath.Paths))
		}
	}
}

func main() {
	devices, _, HBAs, err := sastopo.ScsiDevices()
	if err != nil {
		fmt.Print(err)
	}
	for _, device := range devices {
		fmt.Printf("Found Device: %#v \n", device.HBA)
	}
	fmt.Printf("Found HBAs: %#v \n", HBAs)

	findDevMissingPaths(2, devices)

}
