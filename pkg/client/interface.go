package client

import (
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/senergy-platform-connector/test/client"
)

type Factory = func(authClientId string, authClientSecret string, mqttUrl string, deviceManagerUrl string, deviceRepoUrl string, authUrl string, userName string, password string, hubId string, hubName string, devices []client.DeviceRepresentation) (Client, error)

type Client interface {
	Stop()
	HubId() string
	ListenCommandWithQos(deviceUri string, serviceUri string, qos byte, f func(msg platform_connector_lib.CommandRequestMsg) (resp platform_connector_lib.CommandResponseMsg, err error)) error
	SendEventWithQos(deviceUri string, serviceUri string, event map[platform_connector_lib.ProtocolSegmentName]string, b byte) error
}
