package sastopo

import (
	"strconv"

	"github.com/bensallen/go-sysfs"
)

// Device is a SCSI Generic Device
type Device struct {
	Host       int
	Chan       int
	ID         string
	Type       int
	Vendor     string
	Model      string
	Rev        string
	SasAddress string
	Serial     string
	Slot       int
	Enclosure  *Device
	HBA        *HBA
}

// updateSysfsAttrs adds or updates Model, Vendor, Rev, and SasAddress from sysfs for a SG device
func (d *Device) updateSysfsAttrs(obj sysfs.Object) error {

	model, err := obj.Attribute("model").Read()
	if err != nil {
		return err
	}
	vendor, err := obj.Attribute("vendor").Read()
	if err != nil {
		return err
	}
	rev, err := obj.Attribute("rev").Read()
	if err != nil {
		return err
	}
	sasAddress, err := obj.Attribute("sas_address").Read()
	if err != nil {
		return err
	}

	devType, err := obj.Attribute("type").ReadInt()
	if err != nil {
		return err
	}

	d.Model = model
	d.Vendor = vendor
	d.Rev = rev
	d.SasAddress = sasAddress
	d.Type, _ = devType

	return nil
}

func (d *Device) updateDriveSerial(obj sysfs.Object) error {
	sn, err := vpd80(obj)
	if err != nil {
		return err
	}
	d.Serial = sn
	return nil
}

func (d *Device) updateSerial(obj sysfs.Object) error {
	switch d.Type {
	case 0:
		if err := d.updateDriveSerial(obj); err != nil {
			return err
		}
	case 13:
		if err := d.updateEnclosureSerial(obj); err != nil {
			return err
		}
	default:
		return &errUnknownType{"dev: " + d.ID + " type: " + strconv.Itoa(d.Type)}
	}
	return nil
}

// SgDevices returns map[int]Device of all SG devices
/*func SgDevices(sgDevicesPath string) (map[int]*Device, error) {
	var devices = map[int]*Device{}

	file, err := os.Open(sgDevicesPath)
	if err != nil {
		return devices, err
	}
	r := yacr.NewReader(file, '\t', false, false)
	var Host, Chan, ID, Lun, Type, Opens, Qdepth, Busy, Online int
	for {
		if n, err := r.ScanRecord(&Host, &Chan, &ID, &Lun, &Type, &Opens, &Qdepth, &Busy, &Online); err != nil {
			break
		} else if n != 9 {
			break
		}
		devices[ID] = &Device{
			Host:   Host,
			Chan:   Chan,
			ID:     ID,
			Lun:    Lun,
			Type:   Type,
			Opens:  Opens,
			Qdepth: Qdepth,
			Busy:   itob(Busy),
			Online: itob(Online),
		}
		if err := devices[ID].updateSysfsAttrs(); err != nil {
			return devices, err
		}
		if err := devices[ID].updateSerial(); err != nil {
			return devices, err
		}

	}
	err = file.Close()
	return devices, err
}*/

// SgDevices2 returns map[int]Device of all SG devices
func SgDevices2() (map[string]*Device, error) {
	var devices = map[string]*Device{}
	sysfsObjects := sysfs.Class.Object("scsi_device").SubObjects()

	for d := 0; d <= len(sysfsObjects); d++ {
		devices[sysfsObjects[d].Name()] = &Device{
			ID: sysfsObjects[d].Name(),
		}
		if err := devices[sysfsObjects[d].Name()].updateSysfsAttrs(sysfsObjects[d].SubObject("device")); err != nil {
			return devices, err
		}
		if err := devices[sysfsObjects[d].Name()].updateSerial(sysfsObjects[d].SubObject("device")); err != nil {
			return devices, err
		}
	}
	return devices, nil

}
