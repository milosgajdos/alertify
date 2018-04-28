package alertify

// Monitor monitors some activity and sends message to channel
type Monitor interface {
	// MonitorAndAlert sends messages to message channel
	MonitorAndAlert(chan<- *Msg) error
	// Stop stops the monitor
	Stop()
	// String implements stringer interface
	String() string
}
