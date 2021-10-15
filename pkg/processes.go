package pkg

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel/v2"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/configuration"
	"log"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type Process struct {
	Id string `json:"id"`
}

func EnsureProcesses(ctx context.Context, wg *sync.WaitGroup, config configuration.Config, hubId string) (processes []Process, err error) {
	processes, err = LoadProcesses(config)
	if err != nil {
		processes, err = CreateProcesses(config, hubId)
		if err != nil {
			log.Println("ERROR: unable to create processes")
			return
		}
		err = StoreProcesses(config, processes)
		if err != nil {
			deleteErr := DeleteProcesses(config, processes)
			log.Println("ERROR: unable to store processes", err, deleteErr)
			return
		}
	}
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		<-ctx.Done()
		err := DeleteProcesses(config, processes)
		if err != nil {
			log.Println("ERROR: unable to delete process", err)
		}
		if wg != nil {
			wg.Done()
		}
	}()
	return
}

func CreateProcesses(config configuration.Config, hubId string) (processes []Process, err error) {
	token, err := security.GetOpenidPasswordToken(config.AuthUrl, config.AuthClientId, config.AuthClientSecret, config.UserName, config.Password)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return processes, err
	}
	devices, err := GetHubDeviceIds(config, hubId, token.JwtToken(), false)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return processes, err
	}
	prepared, err := GetPreparedProcess(config, token.JwtToken())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return processes, err
	}
	for i, device := range devices {
		if int64(i)%config.OneProcessEveryNDevices == 0 {
			deployedProcess, err := CreateProcess(config, prepared, device, token.JwtToken())
			if err != nil {
				log.Println("ERROR:", err)
				debug.PrintStack()
				return processes, err
			}
			processes = append(processes, Process{Id: deployedProcess.Id})
		}
	}
	return
}

func DeleteProcesses(config configuration.Config, processes []Process) (err error) {
	openidToken, err := security.GetOpenidPasswordToken(config.AuthUrl, config.AuthClientId, config.AuthClientSecret, config.UserName, config.Password)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}
	token := openidToken.JwtToken()
	for _, p := range processes {
		resp, err := token.Delete(config.ProcessDeploymentUrl + "/v2/deployments/" + url.QueryEscape(p.Id))
		if err != nil {
			log.Println("ERROR:", err)
			debug.PrintStack()
			return err
		}
		resp.Body.Close()
	}
	return nil
}

func StoreProcesses(config configuration.Config, processes []Process) (err error) {
	file, err := os.OpenFile(config.ProcessInfoLocation, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(processes)
}

func LoadProcesses(config configuration.Config) (processes []Process, err error) {
	file, err := os.Open(config.ProcessInfoLocation)
	if err != nil {
		return processes, err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&processes)
	return
}

func GetPreparedProcess(config configuration.Config, token security.JwtToken) (result deploymentmodel.Deployment, err error) {
	err = token.GetJSON(config.ProcessDeploymentUrl+"/v2/prepared-deployments/"+url.QueryEscape(config.ProcessModelId)+"?with_options=false", &result)
	return
}

func CreateProcess(config configuration.Config, prepared deploymentmodel.Deployment, device string, token security.JwtToken) (result deploymentmodel.Deployment, err error) {
	result.Name = device
	for i, element := range prepared.Elements {
		if element.Task != nil {
			element.Task.Selection.SelectedDeviceId = &device
			element.Task.Selection.SelectedServiceId = &config.ProcessServiceId
			prepared.Elements[i] = element
		}
		if element.TimeEvent != nil {
			if element.TimeEvent.Type == "timeDuration" && element.TimeEvent.Time == "" {
				d, err := time.ParseDuration(config.ProcessInterval)
				if err != nil {
					return result, err
				}
				element.TimeEvent.Time = formatIsoDuration(d)
			}
		}
	}
	err = token.PostJSON(config.ProcessDeploymentUrl+"/v2/deployments", prepared, &result)
	return
}

func formatIsoDuration(dur time.Duration) string {
	return "PT" + strings.ToUpper(dur.Truncate(time.Millisecond).String())
}
