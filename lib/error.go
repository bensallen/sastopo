package sastopo

import "errors"

// ErrUnknownType is when a SCSI device is found that isn't a type that we know how to handle
var ErrUnknownType = errors.New("unknown device type")
