package pkg

import (
	"context"
	"encoding/json"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/iot"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/configuration"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/statistics"
	"github.com/SENERGY-Platform/senergy-platform-connector/test/client"
	"log"
	"math/rand"
	"net/url"
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

const DeviceUriKey = "deviceUri"
const ServiceUriKey = "serviceUri"
const ProcessIdKey = "processId"

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
	log.Println("INFO: use", len(devices), "devices; config config.DeviceCount=", config.DeviceCount)
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
	log.Println("started client with id", c.HubId)
	if c.HubId != clientInfo.Id {
		log.Println("store new hub id in file", c.HubId, config.ClientInfoLocation)
		clientInfo.Id = c.HubId
		err = json.NewEncoder(file).Encode(clientInfo)
		if err != nil {
			log.Println("WARNING: unable to encode client info at", config.ClientInfoLocation, err)
			err = nil
		}
		err = file.Sync()
		if err != nil {
			log.Println("WARNING: unable to flush client info to", config.ClientInfoLocation, err)
			err = nil
		}
	}

	var stat statistics.Interface
	if config.StatisticsInterval != "" && config.StatisticsInterval != "-" {
		statisticsInterval, err := time.ParseDuration(config.StatisticsInterval)
		if err != nil {
			log.Println("WARNING: no valid statistics interval")
			stat = statistics.Void{}
		} else {
			stat = statistics.New(ctx, statisticsInterval)
		}
	}

	err = simServices(ctx, config, err, devices, c, stat)
	if err != nil {
		return err
	}
	if config.ProcessModelId != "" {
		processes, err := EnsureProcesses(ctx, wg, config, c.HubId)
		if err != nil {
			log.Println("WARNING: unable to create processes", err)
			return nil
		}
		if config.ProcessInterval != "" && config.ProcessInterval != "-" {
			err = triggerProcesses(ctx, config, processes)
			if err != nil {
				return err
			}
		}
	}
	if config.AnalyticsFlowId != "" {
		_, err = EnsureAnalytics(ctx, wg, config, c.HubId)
		if err != nil {
			log.Println("WARNING: unable to create analytics", err)
			return nil
		}
	}
	return nil
}

func simServices(ctx context.Context, config configuration.Config, err error, devices []client.DeviceRepresentation, c *client.Client, stat statistics.Interface) error {
	messages := make(chan Message, 10000)
	interval, err := time.ParseDuration(config.EmitterInterval)
	if err != nil {
		log.Println("ERROR: unable to parse emitter_interval", config.EmitterInterval, err)
		return err
	}
	for _, d := range devices {
		err = c.ListenCommandWithQos(d.Uri, config.ServiceUri, byte(config.Qos), func(msg platform_connector_lib.CommandRequestMsg) (resp platform_connector_lib.CommandResponseMsg, err error) {
			if config.Debug {
				log.Println("DEBUG: receive command")
			}
			stat.CommandsHandled()
			payload := createPayload(config)
			err = json.Unmarshal([]byte(payload), &resp)
			return
		})
		if err != nil {
			log.Println("ERROR: unable to listen to device command for", d.Uri, config.ServiceUri, err)
			return err
		}
		//create emitter of event messages
		Emitter(ctx, messages, map[string]string{
			DeviceUriKey:  d.Uri,
			ServiceUriKey: config.ServiceUri,
		}, interval, func() string {
			stat.EventEmitted()
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
			start := time.Now()
			err = c.SendEventWithQos(m.Info[DeviceUriKey], m.Info[ServiceUriKey], event, byte(config.Qos))
			if err != nil {
				log.Println("ERROR: unable to send emitted event", m.Message, err)
				continue
			}
			stat.EventProduce(time.Since(start))
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

func triggerProcesses(ctx context.Context, config configuration.Config, processes []Process) (err error) {
	openIdToken, err := security.GetOpenidPasswordToken(config.AuthUrl, config.AuthClientId, config.AuthClientSecret, config.UserName, config.Password)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}
	token := openIdToken.JwtToken()
	interval, err := time.ParseDuration(config.ProcessInterval)
	if err != nil {
		log.Println("ERROR: unable to parse emitter_interval", config.EmitterInterval, err)
		return err
	}

	if config.ProcessStartOnce {
		for _, process := range processes {
			go func(p Process) {
				//wait for random time between now and interval to offset emitter
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				if interval <= 1<<31-1 {
					time.Sleep(time.Duration(r.Int31n(int32(interval))))
				} else {
					time.Sleep(time.Duration(r.Int63n(int64(interval))))
				}
				TriggerProcess(config, p.Id, token)
			}(process)
		}
		return nil
	} else {
		messages := make(chan Message, len(processes))
		for _, process := range processes {
			Emitter(ctx, messages, map[string]string{ProcessIdKey: process.Id}, interval, func() string { return "" })
		}
		//send event messages created by Emitter()
		go func() {
			for m := range messages {
				processId := m.Info[ProcessIdKey]
				TriggerProcess(config, processId, token)
			}
		}()
		return nil
	}
}

func TriggerProcess(config configuration.Config, processId string, token security.JwtToken) {
	resp, err := token.Get(config.ProcessEngineWrapperUrl + "/v2/deployments/" + url.QueryEscape(processId) + "/start")
	if err != nil {
		log.Println("ERROR:", err)
		return
	}
	resp.Body.Close()
}
