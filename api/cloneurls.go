package api

// CloneURLs holds the clone URLs a provider returned for a new repository.
// Either may be empty; the caller chooses one based on the configured protocol.
type CloneURLs struct {
	SSH   string
	HTTPS string
}
