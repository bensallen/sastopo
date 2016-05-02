package scsi

import "testing"

func TestSgDevices(t *testing.T) {
	devices, err := SgDevices("/proc/scsi/sg/devices")
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) == 0 {
		t.Fatal("No values")
	}
}

/*func TestSgAttributes(t *testing.T) {
	attributes := SgAttributes(0)
	if len(attributes) == 0 {
		t.Fatal("No values")
	}
}*/
