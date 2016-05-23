package sastopo

import (
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
	Enclosure  *Device
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

func updateMultiPaths(devices map[string]*Device, devicesBySerial map[string]map[*Device]bool, devicesBySASAddress map[string]map[*Device]bool) map[string]*MultiPathDevice {
	var multiPathDevices = map[string]*MultiPathDevice{}
	for _, device := range devices {

		if devicesBySerial[device.Serial] != nil {
			device.MultiPath = &MultiPathDevice{
				Paths: devicesBySerial[device.Serial],
			}
			multiPathDevices[device.Serial] = device.MultiPath
		} else if devicesBySASAddress[device.SasAddress] != nil {
			device.MultiPath = &MultiPathDevice{
				Paths: devicesBySASAddress[device.SasAddress],
			}
			multiPathDevices[device.SasAddress] = device.MultiPath
		} else {
			log.Printf("Warning: Did not find device: %s, in devicesBySerial or devicesBySASAddress", device.ID)
		}
	}
	return multiPathDevices
}

// ScsiDevices returns map[string]*Device of all SCSI devices and
// map[string]*MultiPathDevice of all resolved unique end devices
func ScsiDevices() (map[string]*Device, map[string]*MultiPathDevice, map[string]*HBA, error) {
	var Devices = map[string]*Device{}
	var DevicesBySerial = map[string]map[*Device]bool{}
	var DevicesBySASAddress = map[string]map[*Device]bool{}
	var HBAs = map[string]*HBA{}
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

		/*if Devices[name].Type == 13 {
			Enclosures[name] = &Enclosure{
				sysfsObj: ,
			}
		}*/
	}
	multiPathDevices := updateMultiPaths(Devices, DevicesBySerial, DevicesBySASAddress)

	return Devices, multiPathDevices, HBAs, nil

}
