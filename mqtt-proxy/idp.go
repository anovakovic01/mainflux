package mqtt

// IdentityProvider has methods necessary for thing authorization.
type IdentityProvider interface {
	// CanAccess checkes if thing has access to given channel.
	CanAccess(chanID, key string) (string, error)

	// Identify identifies thing using token and returns its ID.
	Identify(token string) (string, error)
}
