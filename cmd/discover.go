package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	sastopo "gitlab.alcf.anl.gov/jlse/sastopo/lib"
	yaml "gopkg.in/yaml.v2"
)

var conf sastopo.Conf

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover host's SAS Topology",
	Long:  "Discover host's SAS Topology",
	Run:   run,
}

func init() {
	RootCmd.AddCommand(discoverCmd)
	discoverCmd.Flags().BoolVarP(&conf.Summary, "summary", "s", true, "Show summary of SAS devices")
	discoverCmd.Flags().BoolVarP(&conf.Mismatch, "mismatch", "m", false, "Show devices with path count mismatch")
	discoverCmd.Flags().IntVarP(&conf.PathCount, "pathcount", "p", 2, "Number of expected paths to each SAS device")
	discoverCmd.Flags().IntVar(&conf.SysfsMatchPathEncl, "sysfsMatchPathEncl", 8, "Number of sysfs elements expected for a sysfs device")

}

func run(cmd *cobra.Command, args []string) {
	loadConf()

	devices, multiPathDevices, enclosures, HBAs, err := sastopo.ScsiDevices(conf)
	if err != nil {
		fmt.Print(err)
	}
	if conf.Mismatch {
		findDevMissingPaths(conf.PathCount, devices)
	}
	if conf.Summary {
		summary(devices, multiPathDevices, enclosures, HBAs)
	}
}

func findDevMissingPaths(count int, devices map[string]*sastopo.Device) {
	for _, d := range devices {
		if len(d.MultiPath.Paths) != count {
			fmt.Printf("Path Count Mismatch: %#v, found %d paths\n", d.Serial, len(d.MultiPath.Paths))
		}
	}
}

func summary(devices map[string]*sastopo.Device, multiPathDevices map[string]*sastopo.MultiPathDevice, enclosures map[*sastopo.Enclosure]bool, HBAs map[string]*sastopo.HBA) {

	fmt.Printf("Found %d SAS Devices\n", len(devices))
	//for _, device := range devices {
	//	fmt.Printf("Found Device: %p\n", device.Enclosure)
	//}
	fmt.Printf("Found %d Unique Multi-pathed SAS Devices\n", len(multiPathDevices))
	for hba := range HBAs {
		fmt.Printf("Found HBA: %s, Slot: %s, Host: %s\n", HBAs[hba].PciID, HBAs[hba].Slot, HBAs[hba].Host)
	}

	fmt.Printf("Found %d Enclosures\n", len(enclosures))
	for enclosure := range enclosures {
		fmt.Printf("Found Enclosure: %p with %d slots populated\n", enclosure, len(enclosure.Slots))
		for path := range enclosure.MultiPathDevice.Paths {
			fmt.Printf("HBA: %s, Slot %s, Port: %s\n", path.HBA.PciID, path.HBA.Slot, path.Port)
		}
		//for slot := range enclosure.Slots {
		//	fmt.Printf("%s\n", slot)
		//}
	}
}

func loadConf() {

	var data = []byte(`

# Labels of PCI bus addresses to Slot ID
HBALabels:
  "0000:11:00.0": 'C3'
  "0000:8b:00.0": 'C5'
  "0000:90:00.0": 'C6'

EnclLabels:
  "0000:11:00.0":
  "0000:8b:00.0": 
  "0000:90:00.0": 
`)

	err := yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

}
