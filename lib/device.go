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
		HBAs[p[5]] = &HBA{PciID: p[5], Host: p[6]}
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

// updateMultiPaths iterates through devices finding multiple paths based on devices
// with same serial number or SAS Address
func updateMultiPaths(devices map[string]*Device) map[string]*MultiPathDevice {
	multiPathDevices := map[string]*MultiPathDevice{}

	for _, device1 := range devices {
		if device1.MultiPath == nil {
			device1.MultiPath = new(MultiPathDevice)
			device1.MultiPath.Paths = map[*Device]bool{}
			device1.MultiPath.Paths[device1] = true
		}

		for _, device2 := range devices {
			if device1 == device2 {
				continue
			}

			if device1.SasAddress != "" && device1.SasAddress == device2.SasAddress {
				if device2.MultiPath == nil {
					device2.MultiPath = device1.MultiPath
				}
				device1.MultiPath.Paths[device2] = true
				multiPathDevices[device1.SasAddress] = device1.MultiPath

			} else if device1.Serial != "" && device1.Serial == device2.Serial {
				if device2.MultiPath == nil {
					device2.MultiPath = device1.MultiPath
				}
				device1.MultiPath.Paths[device2] = true
				multiPathDevices[device1.Serial] = device1.MultiPath
			}
		}
	}
	return multiPathDevices
}

// ScsiDevices returns map[string]*Device of all SCSI devices and
// map[string]*MultiPathDevice of all resolved unique end devices
func ScsiDevices() (map[string]*Device, map[string]*MultiPathDevice, map[string]*HBA, error) {
	var devices = map[string]*Device{}
	var HBAs = map[string]*HBA{}

	scsiDeviceObj := sysfs.Class.Object("scsi_device")
	sysfsObjects := scsiDeviceObj.SubObjects()

	for d := 0; d < len(sysfsObjects); d++ {
		devices[sysfsObjects[d].Name()] = &Device{
			ID:       sysfsObjects[d].Name(),
			sysfsObj: sysfsObjects[d].SubObject("device"),
		}
		if err := devices[sysfsObjects[d].Name()].updateSysfsAttrs(); err != nil {
			log.Printf("Warning: %s", err)
		}
		if err := devices[sysfsObjects[d].Name()].updateSerial(); err != nil {
			log.Printf("Warning: %s", err)
		}
		if err := devices[sysfsObjects[d].Name()].updatePathVars(HBAs); err != nil {
			log.Printf("Warning: %s", err)
		}
		if err := devices[sysfsObjects[d].Name()].updateEnclSlot(); err != nil {
			log.Printf("Warning: %s", err)
		}
	}
	multiPathDevices := updateMultiPaths(devices)
	return devices, multiPathDevices, HBAs, nil

}
