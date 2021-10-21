package pkg

import (
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/configuration"
	"log"
	"net/url"
	"runtime/debug"
	"strconv"
)

func Cleanup(config configuration.Config) error {
	token, err := security.GetOpenidPasswordToken(config.AuthUrl, config.AuthClientId, config.AuthClientSecret, config.UserName, config.Password)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}

	//devices
	limit := 200
	temp := []SearchElement{}
	devices := []SearchElement{}
	var after *ListAfter
	for {
		log.Println("LIST DEVICE BATCH")
		err, _ = QueryPermissionsSearch(config, token.JwtToken(), QueryMessage{
			Resource: "devices",
			Find: &QueryFind{
				QueryListCommons: QueryListCommons{
					Limit:  limit,
					Offset: 0,
					After:  after,
					Rights: "r",
				},
			},
		}, &temp)
		if err != nil {
			return err
		}
		devices = append(devices, temp...)
		if len(temp) < limit {
			break
		}
		if len(temp) > 0 {
			after = &ListAfter{
				SortFieldValue: temp[len(temp)-1].Name,
				Id:             temp[len(temp)-1].Id,
			}
		}
		temp = []SearchElement{}
	}
	for _, d := range devices {
		log.Println("DELETE", d.Id, d.Name)
		resp, err := token.JwtToken().Delete(config.DeviceManagerUrl + "/devices/" + url.QueryEscape(d.Id))
		if err != nil {
			return err
		}
		resp.Body.Close()
	}

	//processes
	offset := 0
	tempProcesses := []SearchElement{}
	processes := []SearchElement{}
	for {
		log.Println("LIST PROCESSES BATCH")
		tempProcesses, err = GetProcessDeploymentList(config, token.JwtToken(), map[string][]string{
			"maxResults":  {strconv.Itoa(limit)},
			"firstResult": {strconv.Itoa(offset)},
		})
		if err != nil {
			return err
		}
		processes = append(processes, tempProcesses...)
		if len(tempProcesses) < limit {
			break
		}
		tempProcesses = []SearchElement{}
		offset = offset + limit
	}
	for _, p := range processes {
		log.Println("DELETE", p.Id, p.Name)
		resp, err := token.JwtToken().Delete(config.ProcessDeploymentUrl + "/v2/deployments/" + url.QueryEscape(p.Id))
		if err != nil {
			log.Println("ERROR:", err)
			debug.PrintStack()
			return err
		}
		resp.Body.Close()
	}
	return nil
}

type SearchElement struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func QueryPermissionsSearch(config configuration.Config, token security.JwtToken, query QueryMessage, result interface{}) (err error, code int) {
	err = token.PostJSON(config.PermissionsQueryUrl+"/v3/query", query, result)
	return
}

type QueryMessage struct {
	Resource string         `json:"resource"`
	Find     *QueryFind     `json:"find"`
	ListIds  *QueryListIds  `json:"list_ids"`
	CheckIds *QueryCheckIds `json:"check_ids"`
}
type QueryFind struct {
	QueryListCommons
	Search string     `json:"search"`
	Filter *Selection `json:"filter"`
}

type QueryListIds struct {
	QueryListCommons
	Ids []string `json:"ids"`
}

type QueryCheckIds struct {
	Ids    []string `json:"ids"`
	Rights string   `json:"rights"`
}

type QueryListCommons struct {
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
	After    *ListAfter `json:"after"`
	Rights   string     `json:"rights"`
	SortBy   string     `json:"sort_by"`
	SortDesc bool       `json:"sort_desc"`
}

type ListAfter struct {
	SortFieldValue interface{} `json:"sort_field_value"`
	Id             string      `json:"id"`
}

type QueryOperationType string

const (
	QueryEqualOperation             QueryOperationType = "=="
	QueryUnequalOperation           QueryOperationType = "!="
	QueryAnyValueInFeatureOperation QueryOperationType = "any_value_in_feature"
)

type ConditionConfig struct {
	Feature   string             `json:"feature"`
	Operation QueryOperationType `json:"operation"`
	Value     interface{}        `json:"value"`
	Ref       string             `json:"ref"`
}

type Selection struct {
	And       []Selection     `json:"and"`
	Or        []Selection     `json:"or"`
	Not       *Selection      `json:"not"`
	Condition ConditionConfig `json:"condition"`
}

func GetProcessDeploymentList(config configuration.Config, token security.JwtToken, query url.Values) (result []SearchElement, err error) {
	err = token.GetJSON(config.ProcessEngineWrapperUrl+"/deployment?"+query.Encode(), &result)
	return
}
