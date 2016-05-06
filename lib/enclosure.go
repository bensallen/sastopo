package sastopo

func (d *Device) updateEnclosureSerial() error {
	switch d.Model {
	case "SA4600":
		return nil
	default:
		sn, err := vpd80(d.ID)
		if err != nil {
			return err
		}
		d.Serial = sn
	}

	return nil
}
