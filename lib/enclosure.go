package sastopo

import "github.com/bensallen/go-sysfs"

func (d *Device) updateEnclosureSerial(obj sysfs.Object) error {
	switch d.Model {
	case "SA4600":
		return nil
	default:
		sn, err := vpd80(obj)
		if err != nil {
			return err
		}
		d.Serial = sn
	}

	return nil
}
