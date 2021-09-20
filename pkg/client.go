package pkg

import (
	"context"
	"encoding/json"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/configuration"
	"github.com/SENERGY-Platform/senergy-platform-connector/test/client"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type ClientInfo struct {
	Id string `json:"id"`
}

func Start(ctx context.Context, config configuration.Config) (err error) {
	client.Id = config.AuthClientId
	client.Secret = config.AuthClientSecret

	file, err := os.Open(config.ClientInfoLocation)
	if err != nil {
		return err
	}
	defer file.Close()
	clientInfo := ClientInfo{}
	err = json.NewDecoder(file).Decode(&clientInfo)
	if err != nil {
		log.Println("WARNING: no valid client info stored at", config.ClientInfoLocation, err)
		err = nil
	}
	devices := GetDevices(config)
	c, err := client.New(config.MqttUrl, config.DeviceManagerUrl, config.DeviceRepoUrl, config.AuthUrl, config.UserName, config.Password, clientInfo.Id, config.HubPrefix, devices)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		c.Stop()
	}()
	if c.HubId != clientInfo.Id {
		clientInfo.Id = c.HubId
		err = json.NewEncoder(file).Encode(clientInfo)
		if err != nil {
			log.Println("WARNING: unable to store client info at", config.ClientInfoLocation, err)
			err = nil
		}
	}
	messages := make(chan Message, config.DeviceCount)
	interval, err := time.ParseDuration(config.EmitterInterval)
	if err != nil {
		log.Println("ERROR: unable to parse emitter_interval", config.EmitterInterval, err)
		return err
	}
	for _, d := range devices {
		err = c.ListenCommand(d.Uri, config.ServiceUri, func(msg platform_connector_lib.CommandRequestMsg) (resp platform_connector_lib.CommandResponseMsg, err error) {
			payload := createPayload(config)
			err = json.Unmarshal([]byte(payload), &resp)
			return
		})
		if err != nil {
			log.Println("ERROR: unable to listen to device command for", d.Uri, config.ServiceUri, err)
			return err
		}
		Emitter(ctx, messages, d.Uri, config.ServiceUri, interval, func() string {
			return createPayload(config)
		})
	}
	go func() {
		for m := range messages {
			event := map[platform_connector_lib.ProtocolSegmentName]string{}
			err := json.Unmarshal([]byte(m.Message), &event)
			if err != nil {
				log.Println("ERROR: unable to unmarshal emitted event", m.Message, err)
				continue
			}
			err = c.SendEvent(m.Device, m.Service, event)
			if err != nil {
				log.Println("ERROR: unable to send emitted event", m.Message, err)
				continue
			}
		}
	}()
	return nil
}

func createPayload(config configuration.Config) string {
	return strings.ReplaceAll(config.ServiceMessage, "__TIME_NOW_UNIX_MS__", strconv.FormatInt(time.Now().Unix(), 10))
}