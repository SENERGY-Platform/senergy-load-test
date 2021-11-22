package senergy

import (
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/client"
	senergyclient "github.com/SENERGY-Platform/senergy-platform-connector/test/client"
)

func Factory(authClientId string, authClientSecret string, mqttUrl string, deviceManagerUrl string, deviceRepoUrl string, authUrl string, userName string, password string, hubId string, hubName string, devices []senergyclient.DeviceRepresentation) (result client.Client, err error) {
	senergyclient.Id = authClientId
	senergyclient.Secret = authClientSecret
	c, err := senergyclient.New(mqttUrl, deviceManagerUrl, deviceRepoUrl, authUrl, userName, password, hubId, hubName, devices)
	if err != nil {
		return result, err
	}
	return &Client{c: c}, nil
}

type Client struct {
	c *senergyclient.Client
}

func (this *Client) Stop() {
	this.c.Stop()
}

func (this *Client) HubId() string {
	return this.c.HubId
}

func (this *Client) ListenCommandWithQos(deviceUri string, serviceUri string, qos byte, f func(msg platform_connector_lib.CommandRequestMsg) (resp platform_connector_lib.CommandResponseMsg, err error)) error {
	return this.c.ListenCommandWithQos(deviceUri, serviceUri, qos, f)
}

func (this *Client) SendEventWithQos(deviceUri string, serviceUri string, event map[platform_connector_lib.ProtocolSegmentName]string, qos byte) error {
	return this.c.SendEventWithQos(deviceUri, serviceUri, event, qos)
}
