package pkg

import (
	"context"
	"encoding/json"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/iot"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/configuration"
	"github.com/SENERGY-Platform/senergy-platform-connector/test/client"
	"log"
	"math/rand"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClientInfo struct {
	Id string `json:"id"`
}

func Start(ctx context.Context, wg *sync.WaitGroup, config configuration.Config) (err error) {
	client.Id = config.AuthClientId
	client.Secret = config.AuthClientSecret

	file, err := os.OpenFile(config.ClientInfoLocation, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
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
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		<-ctx.Done()
		cleanup(config, devices, c)
		c.Stop()
		if wg != nil {
			wg.Done()
		}
	}()
	if c.HubId != clientInfo.Id {
		clientInfo.Id = c.HubId
		err = json.NewEncoder(file).Encode(clientInfo)
		if err != nil {
			log.Println("WARNING: unable to store client info at", config.ClientInfoLocation, err)
			err = nil
		}
	}
	err = simServices(ctx, config, err, devices, c)
	if err != nil {
		return err
	}
	return nil
}

func simServices(ctx context.Context, config configuration.Config, err error, devices []client.DeviceRepresentation, c *client.Client) error {
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
		//create emitter of event messages
		Emitter(ctx, messages, d.Uri, config.ServiceUri, interval, func() string {
			return createPayload(config)
		})
	}

	//send event messages created by Emitter()
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

func cleanup(config configuration.Config, devices []client.DeviceRepresentation, c *client.Client) {
	if config.DeleteOnShutdown {
		token, err := security.GetOpenidPasswordToken(config.AuthUrl, config.AuthClientId, config.AuthClientSecret, config.UserName, config.Password)
		if err != nil {
			log.Println("ERROR:", err)
			debug.PrintStack()
			return
		}
		DeleteDevices(config, devices, token.JwtToken())
		DeleteHub(config, c.HubId, token.JwtToken())
	}
	return
}

func DeleteHub(config configuration.Config, id string, token security.JwtToken) {
	err := iot.New(config.DeviceManagerUrl, "", "", "").DeleteHub(id, token)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
}

func createPayload(config configuration.Config) (result string) {
	result = config.ServiceMessage
	result = strings.ReplaceAll(result, "__TIME_NOW_UNIX_MS__", strconv.FormatInt(time.Now().Unix(), 10))
	result = strings.ReplaceAll(result, "__RAND_PERCENT__", strconv.FormatUint(rand.Uint64()%101, 10))
	return
}
