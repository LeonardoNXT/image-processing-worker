package incomingPayloadNormalizer

import (
	"encoding/json"
	"fmt"
	"image-parsing-worker/config"
	"image-parsing-worker/structs"
)

func Exec(value []byte) structs.IncomingPayload {
	var IncomingPayload structs.IncomingPayload

	err := json.Unmarshal(value, &IncomingPayload)

	if err != nil {
		fmt.Println(config.Error().Consumer + " " + "There was an error transforming JSON in struct" + err.Error())
	}

	return IncomingPayload
}
