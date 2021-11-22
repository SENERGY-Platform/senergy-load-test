package mqtt

import (
	"github.com/SENERGY-Platform/platform-connector-lib/iot"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	senergyclient "github.com/SENERGY-Platform/senergy-platform-connector/test/client"
	"log"
)

func (this *Client) provisionDevices(token security.JwtToken) (newDevices bool, err error) {
	iotClient := iot.New(this.deviceManagerUrl, this.deviceRepoUrl, "", "")
	for _, device := range this.devices {
		d, err := iotClient.GetDeviceByLocalId(device.Uri, token)
		if err != nil && err != security.ErrorNotFound {
			log.Println("ERROR: iotClient.DeviceUrlToIotDevice()", err)
			return false, err
		}
		if err == security.ErrorNotFound {
			d, err = this.createIotDevice(device, token)
			if err != nil {
				log.Println("ERROR: iotClient.CreateIotDevice()", err)
				return false, err
			}
			newDevices = true
		}
		this.deviceLocalIdToId[device.Uri] = d.Id
	}
	return newDevices, nil
}

func (this *Client) createIotDevice(representation senergyclient.DeviceRepresentation, token security.JwtToken) (device model.Device, err error) {
	err = token.PostJSON(this.deviceManagerUrl+"/devices", model.Device{LocalId: representation.Uri, DeviceTypeId: representation.IotType, Name: representation.Name}, &device)
	return
}
