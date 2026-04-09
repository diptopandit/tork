package domain

// Exporter defines the port for exporting a slice of tasks to a byte payload.
type Exporter interface {
	Export(tasks []Task) ([]byte, error)
}
