package sastopo

import (
	"io"
	"os"
	"strconv"
)

func vpd80(sg int) (string, error) {
	file, err := os.Open("/sys/class/scsi_generic/sg" + strconv.Itoa(sg) + "/device/vpd_pg80")
	defer file.Close()
	if err != nil {
		return "", err
	}
	line := make([]byte, 128)
	n, err := file.ReadAt(line, 4)
	if err != nil && err != io.EOF {
		return "", err
	} else if n == 0 {
		return "", nil
	}
	return string(line[:n]), nil
}
