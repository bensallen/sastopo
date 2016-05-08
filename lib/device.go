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
	Host       string
	Port       string
	Slot       string
	HBA        *HBA
	OtherPaths map[*Device]bool
	sysfsObj   sysfs.Object
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

func (d *Device) updatePathVars() error {
	p := strings.Split(string(d.sysfsObj), "/")
	d.Host = p[6]
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

func findOtherPaths(devices map[string]*Device) {
	for _, device1 := range devices {
		if device1.OtherPaths == nil {
			device1.OtherPaths = map[*Device]bool{}
		}

		for _, device2 := range devices {
			if device1 == device2 {
				continue
			}

			if (device1.SasAddress != "" && device1.SasAddress == device2.SasAddress) || (device1.Serial != "" && device1.Serial == device2.Serial) {
				if device2.OtherPaths == nil {
					device2.OtherPaths = map[*Device]bool{}
				}
				device1.OtherPaths[device2] = true
				device2.OtherPaths[device1] = true
			}
		}
	}
}

// ScsiDevices returns map[int]Device of all SCSI devices
func ScsiDevices() (map[string]*Device, error) {
	var devices = map[string]*Device{}
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
		if err := devices[sysfsObjects[d].Name()].updatePathVars(); err != nil {
			log.Printf("Warning: %s", err)
		}
		if err := devices[sysfsObjects[d].Name()].updateEnclSlot(); err != nil {
			log.Printf("Warning: %s", err)
		}
	}
	findOtherPaths(devices)
	return devices, nil

}
