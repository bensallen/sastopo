package sastopo

import (
	"io"

	"github.com/bensallen/go-sysfs"
)

func vpd80(obj sysfs.Object) (string, error) {
	vpdPg80, err := obj.Attribute("vpd_pg80")

	line := make([]byte, 128)

	line, n, err := vpdPg80.ReadBytes(4, 128)

	if err != nil && err != io.EOF {
		return "", err
	} else if n == 0 {
		return "", nil
	}
	start, stop := trimPoints(line[:n])

	return string(line[start:stop]), nil
}
