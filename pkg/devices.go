package pkg

import (
	"github.com/SENERGY-Platform/senergy-load-test/pkg/configuration"
	"github.com/SENERGY-Platform/senergy-platform-connector/test/client"
	"strconv"
)

type Device struct {
	LocalId string `json:"local_id"`
}

func GetDevices(config configuration.Config) (devices []client.DeviceRepresentation) {
	prefix := config.HubPrefix
	for i := 0; i < config.DeviceCount; i++ {
		devices = append(devices, client.DeviceRepresentation{
			IotType: config.DeviceType,
			Uri:     prefix + "_" + strconv.Itoa(i),
			Name:    prefix + "_" + strconv.Itoa(i),
		})
	}
	return
}
