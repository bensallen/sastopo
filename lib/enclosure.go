package sastopo

import (
	"log"
	"os"
	"os/exec"
)

// Enclosure is a SCSI Enclosure Device
type Enclosure struct {
	MultiPathDevice *MultiPathDevice
	Slots           map[string]*MultiPathDevice
}

func (d *Device) updateEnclosureSerial() (err error) {
	var sn string

	switch d.Model {
	case "SA4600":
		if sn, err = ddnSA4600EnclosureSerial(d.SG); err != nil {
			return err
		}
	default:
		if sn, err = vpd80(d.sysfsObj); err != nil {
			return err
		}
	}
	d.Serial = sn

	return nil
}

// The DDN SA4600 doesn't support vpd_80 for SN. However SES page
// 0x7 has a vendor specific element [0x8e] that shows a device
// labeled as "SHELF" or "Dragon Enclosure" in page 0x1.
// We use sg_ses --hex output, drop all the whitespace and use
// hex.Decode() to turn it into a useable []byte. Finally we take
// the appropraite offset in the 0x7 page, 2068, and grab 16 bytes
// which makes up the serial number.
// This function requires root privledges and sg3_utils to be installed.
func ddnSA4600EnclosureSerial(sg string) (string, error) {
	var (
		cmdOut []byte
		sn     string
		err    error
		page7  []byte
	)

	if _, err = os.Stat("/dev/" + sg); os.IsNotExist(err) {
		return sn, err
	}

	cmd := "sg_ses"
	args := []string{"--page=0x7", "-I7,0", "--raw", "/dev/" + sg}
	if cmdOut, err = exec.Command(cmd, args...).Output(); err != nil {
		log.Printf("Error, running sg_ses failed: %s", err)
		return sn, err
	}
	page7, n, err := sgSesToBytes(cmdOut)
	if err != nil || n == 0 {
		log.Printf("Error, decoding hex output sg_ses failed, found %d bytes: %s", n, err)
		return sn, err
	}
	//log.Printf("Found %d bytes of data from sg_ses, serial is: %#v", n, string(page7[2068:2084]))
	return string(page7[2068:2084]), nil
}

// Enclosures returns a map of all unique Enclosures based on the input *Device map.
// Also updates the Enclosure device's Enclosure attribute.
func Enclosures(enclMap map[*Device]bool) map[*Enclosure]bool {
	var (
		multiPathDevices = map[*MultiPathDevice]bool{}
		enclosures       = map[*Enclosure]bool{}
	)
	for encl := range enclMap {
		multiPathDevices[encl.MultiPath] = true
	}
	for multiPathDevice := range multiPathDevices {
		enclosure := &Enclosure{MultiPathDevice: multiPathDevice}
		enclosures[enclosure] = true
		for device := range multiPathDevice.Paths {
			device.Enclosure = enclosure
		}
	}
	return enclosures
}
