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
	Slot       string
	MultiPath  *MultiPathDevice
	sysfsObj   sysfs.Object
}

// MultiPathDevice contains Devices which has multiple paths
type MultiPathDevice struct {
	Paths map[*Device]bool
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

	blocks := d.sysfsObj.SubObject("block").SubObjects()
	if len(blocks) > 0 {
		d.Block = blocks[0].Name()
	}

	sgs := d.sysfsObj.SubObject("scsi_generic").SubObjects()
	if len(sgs) > 0 {
		d.SG = sgs[0].Name()
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
		return &errUnknownType{"dev: " + d.ID + " type: " + strconv.Itoa(d.Type)}
	}
	return nil
}

// Update HBA and Port attributes of device using the elements of sysfs path
// of the device
func (d *Device) updatePathVars(HBAs map[string]*HBA) error {
	p := strings.Split(string(d.sysfsObj), "/")
	if len(p) < 8 {
		return errors.New("Unexpected Sysfs path: must have least 8 elements in path: " + string(d.sysfsObj))
	}
	if HBAs[p[5]] != nil {
		d.HBA = HBAs[p[5]]
	} else {
		HBAs[p[5]] = &HBA{
			PciID: p[5],
			Host:  p[6],
		}
		d.HBA = HBAs[p[5]]
	}
	d.Port = p[7]

	return nil
}

// updateEnclSlot updates Slot from sysfs, ex: <device>/enclosure_device:Slot 1
func (d *Device) updateEnclSlot() error {
	files, err := filepath.Glob(string(d.sysfsObj) + "/enclosure_device:*")
	if err != nil || len(files) == 0 {
		return err
	} else if len(files) > 1 {
		log.Printf("Warning: found more than one enclosure_device for dev: %s, using the first one", d.ID)
	}

	path := strings.Split(files[0], "/")
	enclSlot := strings.Split(path[len(path)-1], ":")
	if len(enclSlot) == 2 {
		d.Slot = strings.TrimSpace(enclSlot[1])
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

func updateEnclosure(devices map[string]*Device, enclosures map[*Enclosure]bool) {
	var enclosuresBySysfsPrefix = map[string]*Enclosure{}
	for enclosure := range enclosures {
		for device := range enclosure.MultiPathDevice.Paths {
			path := strings.Split(string(device.sysfsObj), "/")
			enclosuresBySysfsPrefix[strings.Join(path[0:8], "/")] = enclosure
		}
	}
	for _, device := range devices {
		path := strings.Split(string(device.sysfsObj), "/")
		device.Enclosure = enclosuresBySysfsPrefix[strings.Join(path[0:8], "/")]
		if device.Slot != "" && device.Enclosure != nil {
			if device.Enclosure.Slots == nil {
				device.Enclosure.Slots = map[string]*MultiPathDevice{}
			}
			device.Enclosure.Slots[device.Slot] = device.MultiPath
		}
	}
}

// ScsiDevices returns map[string]*Device of all SCSI devices and
// map[string]*MultiPathDevice of all resolved unique end devices
func ScsiDevices() (map[string]*Device, map[string]*MultiPathDevice, map[*Enclosure]bool, map[string]*HBA, error) {
	var (
		Devices             = map[string]*Device{}
		DevicesBySerial     = map[string]map[*Device]bool{}
		DevicesBySASAddress = map[string]map[*Device]bool{}
		HBAs                = map[string]*HBA{}
		EnclMap             = map[*Device]bool{}
	)
	//var Enclosures = map[string]*Enclosure{}

	scsiDeviceObj := sysfs.Class.Object("scsi_device")
	sysfsObjects := scsiDeviceObj.SubObjects()

	for d := 0; d < len(sysfsObjects); d++ {
		name := sysfsObjects[d].Name()
		Devices[name] = &Device{
			ID:       name,
			sysfsObj: sysfsObjects[d].SubObject("device"),
		}
		if err := Devices[name].updateSysfsAttrs(); err != nil {
			log.Printf("Warning: %s", err)
		}
		if err := Devices[name].updateSerial(); err != nil {
			log.Printf("Warning: %s", err)
		}
		if err := Devices[name].updatePathVars(HBAs); err != nil {
			log.Printf("Warning: %s", err)
		}
		if err := Devices[name].updateEnclSlot(); err != nil {
			log.Printf("Warning: %s", err)
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
	updateEnclosure(Devices, enclosures)

	return Devices, multiPathDevices, enclosures, HBAs, nil

}
