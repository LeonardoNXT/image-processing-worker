package structs

type IncomingPayload struct {
	EventID          string        `json:"eventId"`
	ImageID          string        `json:"imageId"`
	OwnerID          string        `json:"ownerId"`
	ObjectKey        string        `json:"objectKey"`
	Service          string        `json:"service"`
	MaxSizeBytes     int           `json:"maxSizeBytes"`
	OutputShape      []OutputShape `json:"outputShape"`
	OutputTopic      string        `json:"outputTopic"`
	PermittedFormats []string      `json:"permittedFormats"`
}

type OutputShape struct {
	MaxWidth  int    `json:"width"`
	MaxHeight int    `json:"height"`
	Format    string `json:"format"`
	Quality   int    `json:"quality"`
}
