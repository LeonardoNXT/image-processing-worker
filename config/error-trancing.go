package config

type ErrorTracing struct {
	Consumer  string
	Messenger string
	Structs   string
}

func Error() ErrorTracing {
	return ErrorTracing{
		Consumer:  "[Error.Consumer]",
		Messenger: "[Error.Messenger]",
		Structs:   "[Error.Structs]",
	}
}
