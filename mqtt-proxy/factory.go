package mqtt

// ServiceFactory represents factory object for mqtt service.
type ServiceFactory interface {
	// New returns new service.
	New() Service
}

type factory struct{}

// NewFactory returns new service.
func NewFactory() ServiceFactory {
	return factory{}
}

func (f factory) New() Service {
	return &service{}
}
