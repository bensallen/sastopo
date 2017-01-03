package sastopo

// HBA is a PCI SAS Host-bus Adapter
type HBA struct {
	PciID string            // PCI Bus ID
	Host  string            // SCSI Host ID
	Slot  string            // Label that describes physical location
	Ports map[*HBAPort]bool // SAS Ports
}

// HBAPort is a HBA Port
type HBAPort struct {
	PortID string        // SCSI HBA Port ID
	Phys   map[*Phy]bool // Map of Phys
}

// Phy is a SAS Phy
type Phy struct {
	PhyIdentifier string //phy_identifier
	SasAddress    string //sas_address
}

func (h *HBA) Port(p string) *HBAPort {
	for port := range h.Ports {
		if port.PortID == p {
			return port
		}
	}
	return nil
}

// PhyIds returns a map of all Phy identifiers on the HBA
func (h *HBA) PhyIds() []string {
	var ids []string

	for port := range h.Ports {
		for phy := range port.Phys {
			ids = append(ids, phy.PhyIdentifier)
		}
	}
	return ids
}

// PhyIds returns a map of all Phy identifiers for the HBAPort
func (p *HBAPort) PhyIds() []string {
	var ids []string

	for phy := range p.Phys {
		ids = append(ids, phy.PhyIdentifier)
	}

	return ids
}
