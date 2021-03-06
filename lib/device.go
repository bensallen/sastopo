package sastopo

import (
	"errors"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bensallen/go-sysfs"
)

// Device is a SCSI Generic Device
type Device struct {
	ID         string
	Type       int
	Vendor     string
	Model      string
	Rev        string
	SasAddress string
	Serial     string
	Block      string
	SG         string
	Enclosure  *Enclosure
	HBA        *HBA
	Port       string
	Slot       int
	MultiPath  *MultiPathDevice
	sysfsObj   sysfs.Object
}

// MultiPathDevice contains Devices which has multiple paths
type MultiPathDevice struct {
	Paths map[*Device]bool
}

// Devices converts the MultiPathDevice Paths map to a slice of *Devices
func (mpd *MultiPathDevice) Devices() []*Device {
	var d []*Device
	for device := range mpd.Paths {
		d = append(d, device)
	}
	return d
}

// Serial returns the first serial attribute from a MultiPathDevice
func (mpd *MultiPathDevice) Serial() string {
	for device := range mpd.Paths {
		return device.Serial
	}
	return ""
}

// Model returns the first model attribute from a MultiPathDevice
func (mpd *MultiPathDevice) Model() string {
	for device := range mpd.Paths {
		return device.Model
	}
	return ""
}

// Vendor returns the first vendor attribute from a MultiPathDevice
func (mpd *MultiPathDevice) Vendor() string {
	for device := range mpd.Paths {
		return device.Vendor
	}
	return ""
}

// updateSysfsAttrs adds or updates Model, Vendor, Rev, and SasAddress from sysfs for a sysfs object
func (d *Device) updateSysfsAttrs() error {

	model, err := d.sysfsObj.Attribute("model").Read()
	if err != nil {
		return err
	}
	d.Model = model

	vendor, err := d.sysfsObj.Attribute("vendor").Read()
	if err != nil {
		return err
	}
	d.Vendor = vendor

	rev, err := d.sysfsObj.Attribute("rev").Read()
	if err != nil {
		return err
	}
	d.Rev = rev

	sasAddress, err := d.sysfsObj.Attribute("sas_address").Read()
	// Some devices won't have a sas_address in sysfs, so just warn on it
	if err != nil {
		log.Printf("Warning, cannot find sas_address: %s", err)
	}
	d.SasAddress = sasAddress

	devType, err := d.sysfsObj.Attribute("type").ReadInt()
	if err != nil {
		return err
	}
	d.Type = devType

	block, err := d.sysfsObj.SubObject("block")
	if err == nil {
		d.Block = block.SubObjects()[0].Name()
	}

	sg, err := d.sysfsObj.SubObject("scsi_generic")
	if err == nil {
		d.SG = sg.SubObjects()[0].Name()
	}

	return nil
}

func (d *Device) updateDriveSerial() error {
	sn, err := vpd80(d.sysfsObj)
	if err != nil {
		return err
	}
	d.Serial = sn
	return nil
}

func (d *Device) updateSerial() error {
	switch d.Type {
	case 0:
		if err := d.updateDriveSerial(); err != nil {
			return err
		}
	case 13:
		if err := d.updateEnclosureSerial(); err != nil {
			return err
		}
	default:
		return ErrUnknownType
	}
	return nil
}

func findHBAPorts(host sysfs.Object) map[*HBAPort]bool {

	var HBAPorts = map[*HBAPort]bool{}
	//fmt.Printf("findHBAPorts SubObjectsFilter: %v\n", host.SubObjectsFilter("port-*"))
	for _, port := range host.SubObjectsFilter("port-*") {
		//fmt.Printf("findHBAPorts port: %v\n", port)

		var phys = map[*Phy]bool{}

		for _, phy := range port.SubObjectsFilter("phy-*") {
			//fmt.Printf("findHBAPorts phy: %v\n", phy)

			sasPhy, err := phy.SubObject("sas_phy")

			if err != nil {
				continue
			}

			sasPhy, err = sasPhy.SubObject(phy.Name())

			if err != nil {
				continue
			}

			phyIdentifier, err := sasPhy.Attribute("phy_identifier").Read()
			//fmt.Printf("findHBAPorts phyIdentifier: %v\n", phyIdentifier)

			if err != nil {
				continue
			}
			sasAddress, err := sasPhy.Attribute("sas_address").Read()
			//fmt.Printf("findHBAPorts sasAddress: %v\n", sasAddress)
			if err != nil {
				continue
			}

			phys[&Phy{
				PhyIdentifier: phyIdentifier,
				SasAddress:    sasAddress,
			}] = true

		}
		HBAPorts[&HBAPort{
			PortID: port.Name(),
			Phys:   phys,
		}] = true

	}

	return HBAPorts
}

// Update HBA and Port attributes of device using the elements of sysfs path
// of the device
func (d *Device) updatePathVars(HBAs map[string]*HBA, conf Conf) error {
	p := strings.Split(string(d.sysfsObj), "/")
	if len(p) < 8 {
		return errors.New("Unexpected Sysfs path: must have least 8 elements in path: " + string(d.sysfsObj))
	}
	if HBAs[p[5]] != nil {
		d.HBA = HBAs[p[5]]
	} else {
		host := d.sysfsObj.Parent(-7)
		//fmt.Printf("updatePathVars host: %v\n", host)

		HBAs[p[5]] = &HBA{
			PciID: p[5],
			Host:  p[6],
			Slot:  conf.HBALabels[p[5]],
			Ports: findHBAPorts(host),
		}
		d.HBA = HBAs[p[5]]
	}
	d.Port = p[7]

	return nil
}

// updateEnclSlot updates Slot from sysfs, ex: <device>/enclosure_device:Slot 1 or
// contents of parent end_device's bay_identifier
func (d *Device) updateEnclSlot() error {
	// Only continue on type 0 devices (disks)
	if d.Type != 0 {
		return nil
	}

	// Traverse up in path to disk's end_device
	endDevice := d.sysfsObj.Parent(2)
	// Newer kernels (RHEL 7.3) have bay_identifer in sysfs

	sasDevice, err1 := endDevice.SubObject("sas_device/" + endDevice.Name())

	slot, err2 := sasDevice.Attribute("bay_identifier").ReadInt()
	//fmt.Printf("Vendor: %s, Model: %s, endDevice: %s, sasDevice: %s, slot %d\n", d.Vendor, d.Model, endDevice, sasDevice, slot)

	if err1 == nil && err2 == nil {
		d.Slot = slot
	} else {

		// Older sysfs implmentations exposed slots directly in the target as a symlink named
		// "enclosure_device:<slot name>". Try to parse slot out of that path.
		files, err := filepath.Glob(string(d.sysfsObj) + "/enclosure_device:*")

		if err != nil || len(files) == 0 {
			return err
		} else if len(files) > 1 {
			log.Printf("Warning: found more than one enclosure_device for dev: %s, using the first one", d.ID)
		}

		path := strings.Split(files[0], "/")
		enclSlot := strings.Split(path[len(path)-1], ":")
		if len(enclSlot) == 2 {
			d.Slot, err = strconv.Atoi(strings.TrimSpace(enclSlot[1]))
			return err
		}
	}
	return nil
}

// GetUniqID first tries to return the serial number, if it doesn't exist it falls back to
// SASAddress. If neither exist an empty string is returned and an error.
/*func (d *Device) GetUniqID() (string, error) {
	if d.Serial != "" {
		return d.Serial, nil
	} else if d.SasAddress != "" {
		return d.SasAddress, nil
	}
	return "", errors.New("Serial and SasAddress not found")

}*/

func updateMultiPaths(devices map[string]*Device, devicesBySerial map[string]map[*Device]bool, devicesBySASAddress map[string]map[*Device]bool) map[string]*MultiPathDevice {
	var multiPathDevices = map[string]*MultiPathDevice{}
	for _, device := range devices {
		var multiPath *MultiPathDevice
		var uniqDevice map[*Device]bool
		var id string
		if devicesBySerial[device.Serial] != nil {
			uniqDevice = devicesBySerial[device.Serial]
			id = device.Serial
		} else if devicesBySASAddress[device.SasAddress] != nil {
			uniqDevice = devicesBySerial[device.SasAddress]
			id = device.SasAddress
		} else {
			log.Printf("Warning: Did not find device: %s, in devicesBySerial or devicesBySASAddress", device.ID)
			continue
		}

		if multiPathDevices[id] == nil {
			multiPath = &MultiPathDevice{
				Paths: uniqDevice,
			}
		} else {
			multiPath = multiPathDevices[id]
		}
		device.MultiPath = multiPath
		multiPathDevices[device.Serial] = multiPath

	}
	return multiPathDevices
}

// updateEnclosure iterates through all input devices, and compares the
// sysfs path to the input enclosures. We look at the first n element in
// the path. An enclosure's path will match the path of its devices typically
// till the SCSI port or first expander, Example of the latter:
// /sys/devices/pci0000:80/0000:80:03.0/0000:90:00.0/host2/port-2:0/expander-2:0
func updateEnclosure(devices map[string]*Device, enclosures map[*Enclosure]bool, n int) {
	var enclosuresBySysfsPrefix = map[string]*Enclosure{}
	for enclosure := range enclosures {
		for device := range enclosure.MultiPathDevice.Paths {
			path := strings.Split(string(device.sysfsObj), "/")
			enclosuresBySysfsPrefix[strings.Join(path[0:n], "/")] = enclosure
		}
	}
	for _, device := range devices {
		path := strings.Split(string(device.sysfsObj), "/")
		device.Enclosure = enclosuresBySysfsPrefix[strings.Join(path[0:n], "/")]
		if device.Enclosure != nil {
			if device.Enclosure.Slots == nil {
				device.Enclosure.Slots = map[int]*MultiPathDevice{}
			}
			// Only assign disks (type 0) to slots
			if device.Type == 0 {
				device.Enclosure.Slots[device.Slot] = device.MultiPath
			}
		}
	}
}

// ScsiDevices returns map[string]*Device of all SCSI devices and
// map[string]*MultiPathDevice of all resolved unique end devices.
// Takes an int that specifies how many elements of the devices
// and enclosure sysfs path to match against to assign a device
// to an enclosure
func ScsiDevices(conf Conf) (map[string]*Device, map[string]*MultiPathDevice, map[*Enclosure]bool, map[string]*HBA, error) {
	var (
		Devices             = map[string]*Device{}
		DevicesBySerial     = map[string]map[*Device]bool{}
		DevicesBySASAddress = map[string]map[*Device]bool{}
		HBAs                = map[string]*HBA{}
		EnclMap             = map[*Device]bool{}
	)

	scsiDeviceObj := sysfs.Class.Object("scsi_device")
	sysfsObjects := scsiDeviceObj.SubObjects()

	for d := 0; d < len(sysfsObjects); d++ {
		name := sysfsObjects[d].Name()
		sysfsObj, err := sysfsObjects[d].SubObject("device")
		if err != nil {
			log.Printf("Warning, %s, skipping device %s", err, name)
			continue
		}
		Devices[name] = &Device{
			ID:       name,
			sysfsObj: sysfsObj,
			Type:     -1,
		}
		if err := Devices[name].updateSysfsAttrs(); err != nil {
			log.Printf("Warning: %s", err)
		}
		if err := Devices[name].updateSerial(); err != nil {
			if err == ErrUnknownType {
				delete(Devices, name)
				log.Printf("Warning, %s, skipping device %s", err, name)
				continue
			} else if err != nil {
				log.Printf("Warning: %s", err)
			}
		}
		if err := Devices[name].updatePathVars(HBAs, conf); err != nil {
			log.Printf("Warning: %s", err)
		}

		if Devices[name].Type == 0 {
			if err := Devices[name].updateEnclSlot(); err != nil {
				log.Printf("Warning: %s", err)
			}
		}

		// Populate DevicesBySerial
		if Devices[name].Serial != "" {
			if DevicesBySerial[Devices[name].Serial] == nil {
				DevicesBySerial[Devices[name].Serial] = map[*Device]bool{}
			}
			DevicesBySerial[Devices[name].Serial][Devices[name]] = true
		}

		// Populate DevicesBySASAddress
		if Devices[name].SasAddress != "" {
			if DevicesBySASAddress[Devices[name].SasAddress] == nil {
				DevicesBySASAddress[Devices[name].SasAddress] = map[*Device]bool{}
			}
			DevicesBySASAddress[Devices[name].SasAddress][Devices[name]] = true
		}

		// Populate EnclMap
		if Devices[name].Type == 13 {
			EnclMap[Devices[name]] = true
		}
	}
	// Assign MultiPathDevice to Devices, get back map of all MultiPath Devices
	multiPathDevices := updateMultiPaths(Devices, DevicesBySerial, DevicesBySASAddress)
	enclosures := Enclosures(EnclMap)
	updateEnclosure(Devices, enclosures, conf.SysfsMatchPathEncl)

	return Devices, multiPathDevices, enclosures, HBAs, nil

}
