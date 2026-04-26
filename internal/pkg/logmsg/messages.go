package logmsg

const (
	AppInitializing = "application_initializing"
	AppStarting     = "starting_application"
	AppRuntimeError = "application_runtime_error"

	InitBusFailed       = "init_bus_failed"
	BusConnectionFailed = "bus_connection_failed"
	InitBusStreamFailed = "init_bus_stream_failed"

	LoadConfigFailed       = "load_config_failed"
	ValidatingConfigFailed = "validating_configuration_failed"

	MarshallingFailed   = "marshalling_failed"
	UnmarshallingFailed = "unmarshalling_failed"
)
