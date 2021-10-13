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
	"encoding/json"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/analytics/model"
	"strconv"
)

func (this *Analytics) GetPipelinesByDeploymentId(token security.JwtToken, deploymentId string) (pipelineIds []string, err error) {
	pipelineIds = []string{}
	pipelines, err := this.GetPipelines(token)
	if err != nil {
		return pipelineIds, err
	}
	for _, pipeline := range pipelines {
		desc := model.EventPipelineDescription{}
		err = json.Unmarshal([]byte(pipeline.Description), &desc)
		if err != nil {
			//candidate does not use event pipeline description format -> is not event pipeline -> is not searched pipeline
			err = nil
			continue
		}
		if desc.DeploymentId == deploymentId {
			pipelineIds = append(pipelineIds, pipeline.Id.String())
		}
	}
	return pipelineIds, nil
}

func (this *Analytics) GetPipelines(token security.JwtToken) (pipelines []model.Pipeline, err error) {
	limit := 500
	offset := 0
	for {
		temp, err := this.getSomePipelines(token, limit, offset)
		if err != nil {
			return pipelines, err
		}
		if temp != nil {
			pipelines = append(pipelines, temp...)
		}
		if len(temp) < limit {
			return pipelines, nil
		} else {
			offset = offset + limit
		}
	}
}

func (this *Analytics) getSomePipelines(token security.JwtToken, limit int, offset int) (pipelines []model.Pipeline, err error) {
	err = token.GetJSON(this.config.PublicPipelineRepoUrl+"/pipeline?limit="+strconv.Itoa(limit)+"&offset="+strconv.Itoa(offset), &pipelines)
	return
}
