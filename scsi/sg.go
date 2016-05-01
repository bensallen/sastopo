package scsi

import (
	"os"
	"strconv"

	"github.com/gwenn/yacr"
)

// Device is a SCSI Generic Device
type Device struct {
	Host         int
	Chan         int
	ID           int
	Lun          int
	Type         int
	Opens        int
	Qdepth       int
	Busy         bool
	Online       bool
	sysfsSgAttrs sysfsSgAttrs
}

type sysfsSgAttrs struct {
	model       string
	vendor      string
	rev         string
	sas_address string
}

func itob(i int) bool {
	if i == 0 {
		return false
	}
	return true
}

func getsgattrbs(sg int, devices map[int]Device) error {

	for {

	}
	sysSgPath := "/sys/class/scsi_generic/sg" + strconv.Itoa(sg) + "/device/"
	if file, err := os.Open(sysSgPath + "sas_address"); err != nil {
		return err
	}
}

// SgDevices returns map[int]Device of all SG devices
func SgDevices(sgDevicesPath string) (map[int]Device, error) {
	var devices = map[int]Device{}

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
		devices[ID] = Device{
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
	}
	err = file.Close()
	return devices, err
}
