package sastopo

// HBA is a PCI SAS Host-bus Adapter
type HBA struct {
	PciID string            // PCI Bus ID
	Host  string            // SCSI Host ID
	Slot  string            // Label that describes physical location
	Ports map[*HBAPort]bool // SAS Ports
}

// HBAPort is a a HBA Port
type HBAPort struct {
	PortID string        // SCSI HBA Port ID
	Phys   map[*Phy]bool // Map of Phys
}

// Phy is SAS Phy
type Phy struct {
	PhyIdentifier string
	SasAddress    string
}
