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
	devices, multiPathDevices, enclosures, hbas, err := sastopo.ScsiDevices(8)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Printf("Found %d Devices\n", len(devices))
	//for _, device := range devices {
	//	fmt.Printf("Found Device: %p\n", device.Enclosure)
	//}
	fmt.Printf("Found %d Unique Multi-pathed Devices\n", len(multiPathDevices))
	fmt.Printf("Found %d HBAs\n", len(hbas))

	fmt.Printf("Found %d Enclosures\n", len(enclosures))
	for enclosure := range enclosures {
		fmt.Printf("Found Enclosure: %p with %d slots populated\n", enclosure, len(enclosure.Slots))
		//for slot := range enclosure.Slots {
		//	fmt.Printf("%s\n", slot)
		//}
	}
	//findDevMissingPaths(2, devices)

}
