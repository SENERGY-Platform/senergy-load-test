package mqtt

import (
	"errors"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/client"
	senergyclient "github.com/SENERGY-Platform/senergy-platform-connector/test/client"
	paho "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/satori/go.uuid"
	"log"
	"sync"
	"time"
)

func Factory(authClientId string, authClientSecret string, mqttUrl string, deviceManagerUrl string, deviceRepoUrl string, authUrl string, userName string, password string, hubId string, hubName string, devices []senergyclient.DeviceRepresentation) (result client.Client, err error) {
	log.Println("mqtt client is used --> no hub will be created --> HubId == \"\"")
	c := &Client{
		authUrl:          authUrl,
		mqttUrl:          mqttUrl,
		deviceManagerUrl: deviceManagerUrl,
		deviceRepoUrl:    deviceRepoUrl,
		authClientId:     authClientId,
		password:         password,
		devices:          devices,
		userName:         userName,
		authClientSecret: authClientSecret,

		deviceLocalIdToId: map[string]string{},

		mqttClientId:  uuid.NewV4().String(),
		subscriptions: map[string]Subscription{},
	}

	token, err := security.GetOpenidPasswordToken(authUrl, authClientId, authClientSecret, userName, password)
	if err != nil {
		return result, err
	}

	newDevices, err := c.provisionDevices(token.JwtToken())
	if err != nil {
		return result, err
	}
	if newDevices {
		time.Sleep(10 * time.Second) //wait for device creation
	}
	err = c.startMqtt()
	return c, err
}

type Client struct {
	authClientId     string
	authClientSecret string
	mqttUrl          string
	deviceManagerUrl string
	deviceRepoUrl    string
	authUrl          string
	userName         string
	password         string
	devices          []senergyclient.DeviceRepresentation

	mqtt             paho.Client
	subscriptionsMux sync.Mutex
	subscriptions    map[string]Subscription

	deviceLocalIdToId map[string]string
	mqttClientId      string
}

func (this *Client) Stop() {
	this.mqtt.Disconnect(0)
}

func (this *Client) SendEventWithQos(deviceUri string, serviceUri string, event map[platform_connector_lib.ProtocolSegmentName]string, qos byte) error {
	topic := "event/" + this.deviceLocalIdToId[deviceUri] + "/" + serviceUri
	return this.PublishStr(topic+"/resp", event["data"], qos)
}

func (this *Client) ListenCommandWithQos(deviceUri string, serviceUri string, qos byte, handler func(msg platform_connector_lib.CommandRequestMsg) (platform_connector_lib.CommandResponseMsg, error)) error {
	if !this.mqtt.IsConnected() {
		log.Println("WARNING: mqtt client not connected")
		return errors.New("mqtt client not connected")
	}
	topic := "command/" + this.deviceLocalIdToId[deviceUri] + "/" + serviceUri
	callback := func(client paho.Client, message paho.Message) {
		respMsg, err := handler(map[platform_connector_lib.ProtocolSegmentName]string{"data": string(message.Payload())})
		if err != nil {
			log.Println("ERROR: while processing command", err)
			return
		}
		go func() {
			err = this.PublishStr(topic+"/resp", respMsg["data"], qos)
			if err != nil {
				log.Println("ERROR: unable to Publish response", err)
			}
		}()
	}
	token := this.mqtt.Subscribe(topic, qos, callback)
	if token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Subscribe(): ", token.Error())
		return token.Error()
	}
	this.registerSubscription(topic, qos, callback)
	return nil
}

func (this *Client) HubId() string {
	return ""
}
