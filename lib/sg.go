package sastopo

import (
	"os"
	"strconv"

	"github.com/gwenn/yacr"
	"github.com/ungerik/go-sysfs"
)

// Device is a SCSI Generic Device
type Device struct {
	Host       int
	Chan       int
	ID         int
	Lun        int
	Type       int
	Opens      int
	Qdepth     int
	Busy       bool
	Online     bool
	Vendor     string
	Model      string
	Rev        string
	SasAddress string
	Serial     string
	Slot       int
	Enclosure  *Device
	HBA        *HBA
}

func itob(i int) bool {
	if i == 0 {
		return false
	}
	return true
}

// updateSysfsAttrs adds or updates Model, Vendor, Rev, and SasAddress from sysfs for a SG device
func (d *Device) updateSysfsAttrs() error {
	sysfsObject := sysfs.Class.Object("scsi_generic").SubObject("sg" + strconv.Itoa(d.ID)).SubObject("device")

	model, err := sysfsObject.Attribute("model").Read()
	if err != nil {
		return err
	}
	vendor, err := sysfsObject.Attribute("vendor").Read()
	if err != nil {
		return err
	}
	rev, err := sysfsObject.Attribute("rev").Read()
	if err != nil {
		return err
	}
	sasAddress, err := sysfsObject.Attribute("sas_address").Read()
	if err != nil {
		return err
	}

	d.Model = model
	d.Vendor = vendor
	d.Rev = rev
	d.SasAddress = sasAddress

	return nil
}

func (d *Device) updateDriveSerial() error {
	sn, err := vpd80(d.ID)
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
		return &errUnknownType{"dev: /dev/sg" + strconv.Itoa(d.ID) + " type: " + strconv.Itoa(d.Type)}
	}
	return nil
}

// SgDevices returns map[int]Device of all SG devices
func SgDevices(sgDevicesPath string) (map[int]*Device, error) {
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
}
