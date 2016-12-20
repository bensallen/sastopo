package sastopo

// Conf is a struct used for parsing the yaml configure file
type Conf struct {
	Mismatch           bool
	PathCount          int
	SysfsMatchPathEncl int
	Summary            bool
	HBALabels          map[string]string            `yaml:"HBALabels"`
	EnclLabels         map[string]map[string]string `yaml:"EnclLabels"`
}
