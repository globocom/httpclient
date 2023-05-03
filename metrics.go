package httpclient

type Metrics interface {
	// IncrCounter increments the counter value identified by the given name.
	IncrCounter(name string)
	// PushToSeries adds a new value to a histogram identified by the given name.
	PushToSeries(name string, value float64)
	// IncrCounterWithAttrs increments the counter value identified by the given name while adding attributes.
	IncrCounterWithAttrs(name string, attributes map[string]string)
}
