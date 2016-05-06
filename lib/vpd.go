package sastopo

import (
	"io"
	"os"
	"strconv"
)

// trimPoints loops through a []byte looking for left trim and right trim points
// for trimming 0x00 (null) and 0x20 (ascii space)
func trimPoints(line []byte) (start int, stop int) {

	//left trim point
	for i := 0; i < len(line); i++ {
		if line[i] == 0x20 || line[i] == 0x00 {
			continue
		} else {
			start = i
			break
		}
	}

	//right trim point
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == 0x20 || line[i] == 0x00 {
			continue
		} else {
			stop = i + 1
			break
		}
	}
	return start, stop
}

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
	start, stop := trimPoints(line[:n])

	return string(line[start:stop]), nil
}
