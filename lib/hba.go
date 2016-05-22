package sastopo

// HBA is a PCI SAS Host-bus Adapter
type HBA struct {
	PciID string // PCI Bus ID
	Host  string // SCSI Host ID
	Slot  string // Label that describes physical location
}
