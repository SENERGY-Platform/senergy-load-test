package pkg

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/analytics"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/configuration"
	"log"
	"os"
	"runtime/debug"
	"sync"
)

type Analytic struct {
	Id string `json:"id"`
}

func EnsureAnalytics(ctx context.Context, wg *sync.WaitGroup, config configuration.Config, hubId string) (analytics []Analytic, err error) {
	analytics, err = LoadAnalytics(config)
	if err != nil {
		analytics, err = CreateAnalytics(config, hubId)
		if err != nil {
			log.Println("ERROR: unable to create analytics")
			return
		}
		err = StoreAnalytics(config, analytics)
		if err != nil {
			deleteErr := DeleteAnalytics(config, analytics)
			log.Println("ERROR: unable to store analytics", err, deleteErr)
			return
		}
	}
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		<-ctx.Done()
		err := DeleteAnalytics(config, analytics)
		if err != nil {
			log.Println("ERROR: unable to delete analytic", err)
		}
		if wg != nil {
			wg.Done()
		}
	}()
	return
}

func DeleteAnalytics(config configuration.Config, list []Analytic) interface{} {
	openidToken, err := security.GetOpenidPasswordToken(config.AuthUrl, config.AuthClientId, config.AuthClientSecret, config.UserName, config.Password)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}
	token := openidToken.JwtToken()
	a := analytics.New(config)
	for _, pipeline := range list {
		a.Remove(token, pipeline.Id)
	}
	return nil
}

func CreateAnalytics(config configuration.Config, hubId string) (result []Analytic, err error) {
	token, err := security.GetOpenidPasswordToken(config.AuthUrl, config.AuthClientId, config.AuthClientSecret, config.UserName, config.Password)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return result, err
	}
	devices, err := GetHubDeviceIds(config, hubId, token.JwtToken(), false)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return result, err
	}
	a := analytics.New(config)
	for i, device := range devices {
		if int64(i)%config.OneAnalyticsEveryNDevices == 0 {
			pipelineId, err := a.Deploy(token.JwtToken(), device, config.AnalyticsFlowId, device, config.ProcessServiceId)
			if err != nil {
				log.Println("ERROR:", err)
				debug.PrintStack()
				return result, err
			}
			result = append(result, Analytic{Id: pipelineId})
		}
	}
	return
}

func StoreAnalytics(config configuration.Config, analytics []Analytic) (err error) {
	file, err := os.OpenFile(config.AnalyticInfoLocation, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(analytics)
}

func LoadAnalytics(config configuration.Config) (analytics []Analytic, err error) {
	file, err := os.Open(config.AnalyticInfoLocation)
	if err != nil {
		return analytics, err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&analytics)
	return
}
