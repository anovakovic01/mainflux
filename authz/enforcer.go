package authz

// Enforcer contains API for managing and enforcing policies.
type Enforcer interface {
	Enforce(...interface{}) (bool, error)
	AddPolicy(...interface{}) (bool, error)
	RemovePolicy(...interface{}) (bool, error)
	RemoveFilteredPolicy(int, ...string) (bool, error)
}
