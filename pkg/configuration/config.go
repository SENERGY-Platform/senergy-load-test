/*
 * Copyright 2019 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package configuration

import (
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/analytics/model"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	AuthUrl            string `json:"auth_url"`
	AuthClientId       string `json:"auth_client_id"`
	AuthClientSecret   string `json:"auth_client_secret"`
	UserName           string `json:"user_name"`
	Password           string `json:"password"`
	MqttUrl            string `json:"mqtt_url"`
	DeviceManagerUrl   string `json:"device_manager_url"`
	DeviceRepoUrl      string `json:"device_repo_url"`
	DeviceType         string `json:"device_type"`
	DeviceCount        int64  `json:"device_count"`
	ClientInfoLocation string `json:"client_info_location"`
	HubPrefix          string `json:"hub_prefix"`
	ServiceUri         string `json:"service_uri"`
	EmitterInterval    string `json:"emitter_interval"`
	ServiceMessage     string `json:"service_message"`
	DeleteOnShutdown   bool   `json:"delete_on_shutdown"`

	ProcessStartOnce        bool   `json:"process_start_once"`
	ProcessInfoLocation     string `json:"process_info_location"`
	ProcessDeploymentUrl    string `json:"process_deployment_url"`
	ProcessEngineWrapperUrl string `json:"process_engine_wrapper_url"`
	ProcessModelId          string `json:"process_model_id"`
	ProcessServiceId        string `json:"process_service_id"`
	ProcessInterval         string `json:"process_interval"`
	StatisticsInterval      string `json:"statistics_interval"`
	OneProcessEveryNDevices int64  `json:"one_process_every_n_devices"`
	Qos                     int64  `json:"qos"`

	AnalyticInfoLocation      string             `json:"analytic_info_location"`
	PublicFlowEngineUrl       string             `json:"public_flow_engine_url"`
	PublicFlowParserUrl       string             `json:"public_flow_parser_url"`
	PublicPipelineRepoUrl     string             `json:"public_pipeline_repo_url"`
	AnalyticsInputValues      []model.NodeValue  `json:"analytics_input_values"`
	AnalyticsNodeConfig       []model.NodeConfig `json:"analytics_node_config"`
	AnalyticsFlowId           string             `json:"analytics_flow_id"`
	OneAnalyticsEveryNDevices int64              `json:"one_analytics_every_n_devices"`
}

//loads config from json in location and used environment variables (e.g ZookeeperUrl --> ZOOKEEPER_URL)
func LoadConfig(location string) (config Config, err error) {
	file, err := os.Open(location)
	if err != nil {
		return config, err
	}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return config, err
	}
	handleEnvironmentVars(&config)
	return config, nil
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func fieldNameToEnvName(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToUpper(strings.Join(a, "_"))
}

// preparations for docker
func handleEnvironmentVars(config *Config) {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()
	for index := 0; index < configType.NumField(); index++ {
		fieldName := configType.Field(index).Name
		envName := fieldNameToEnvName(fieldName)
		envValue := os.Getenv(envName)
		if envValue != "" {
			fmt.Println("use environment variable: ", envName, " = ", envValue)
			if configValue.FieldByName(fieldName).Kind() == reflect.Int64 {
				i, _ := strconv.ParseInt(envValue, 10, 64)
				configValue.FieldByName(fieldName).SetInt(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Bool {
				b, _ := strconv.ParseBool(envValue)
				configValue.FieldByName(fieldName).SetBool(b)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Float64 {
				f, _ := strconv.ParseFloat(envValue, 64)
				configValue.FieldByName(fieldName).SetFloat(f)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Map {
				value := map[string]string{}
				for _, element := range strings.Split(envValue, ",") {
					keyVal := strings.Split(element, ":")
					key := strings.TrimSpace(keyVal[0])
					val := strings.TrimSpace(keyVal[1])
					value[key] = val
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}
		}
	}
}
