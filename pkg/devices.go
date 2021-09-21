package pkg

import (
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/configuration"
	"github.com/SENERGY-Platform/senergy-platform-connector/test/client"
	"log"
	"net/url"
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

func DeleteDevices(config configuration.Config, devices []client.DeviceRepresentation, token security.JwtToken) {
	for _, d := range devices {
		err := DeleteDevice(config, d.Uri, token)
		if err != nil {
			log.Println("ERROR: ", err)
		}
	}
}

func DeleteDevice(config configuration.Config, id string, token security.JwtToken) (err error) {
	resp, err := token.Delete(config.DeviceManagerUrl + "/local-devices/" + url.QueryEscape(id))
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
