/*
 * Copyright 2020 InfAI (CC SES)
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

package analytics

import (
	"errors"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/analytics/model"
	"log"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

func (this *Analytics) Deploy(token security.JwtToken, label string, flowId string, deviceId string, serviceId string) (pipelineId string, err error) {
	retries := 20
	for i := 0; i < retries; i++ {
		pipelineId, err = this.deploy(token, label, flowId, deviceId, serviceId)
		if err == nil {
			return pipelineId, err
		}
		time.Sleep(1 * time.Second)
	}
	return pipelineId, err
}

func (this *Analytics) deploy(token security.JwtToken, label string, flowId string, deviceId string, serviceId string) (pipelineId string, err error) {
	flowCells, err := this.GetFlowInputs(token, flowId)
	if err != nil {
		log.Println("ERROR: unable to get flow inputs", err.Error())
		debug.PrintStack()
		return "", err
	}
	if len(flowCells) != 1 {
		err = errors.New("expect flow to have exact one operator")
		log.Println("ERROR: ", err.Error())
		debug.PrintStack()
		return "", err
	}

	pipeline, err := this.sendDeployRequest(token, model.PipelineRequest{
		FlowId:      flowId,
		Name:        label,
		Description: "load-test",
		WindowTime:  0,
		Nodes: []model.PipelineNode{
			{
				NodeId: flowCells[0].Id,
				Inputs: []model.NodeInput{{
					FilterIds:  deviceId,
					FilterType: model.DeviceFilterType,
					TopicName:  ServiceIdToTopic(serviceId),
					Values:     this.config.AnalyticsInputValues,
				}},
				Config: this.config.AnalyticsNodeConfig,
			},
		},
	})
	if err != nil {
		log.Println("ERROR: unable to deploy pipeline", err.Error())
		debug.PrintStack()
		return "", err
	}
	pipelineId = pipeline.Id.String()
	return pipelineId, nil
}

func ServiceIdToTopic(id string) string {
	id = strings.ReplaceAll(id, "#", "_")
	id = strings.ReplaceAll(id, ":", "_")
	return id
}

func (this *Analytics) Remove(token security.JwtToken, pipelineId string) (err error) {
	retries := 20
	for i := 0; i < retries; i++ {
		err = this.remove(token, pipelineId)
		if err == nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return err
}

func (this *Analytics) remove(token security.JwtToken, pipelineId string) error {
	resp, err := token.Delete(this.config.PublicFlowEngineUrl + "/pipeline/" + url.PathEscape(pipelineId))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		debug.PrintStack()
		return errors.New("unexpected statuscode")
	}
	return nil
}

func (this *Analytics) sendDeployRequest(token security.JwtToken, request model.PipelineRequest) (result model.Pipeline, err error) {
	err = token.PostJSON(this.config.PublicFlowEngineUrl+"/pipeline", request, &result)
	return
}
