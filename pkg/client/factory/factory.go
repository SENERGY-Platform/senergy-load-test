package factory

import (
	"errors"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/client"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/client/mqtt"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/client/senergy"
)

type ConnectorType int

const (
	Senergy ConnectorType = iota
	Mqtt
)

func Get(connector ConnectorType) client.Factory {
	switch connector {
	case Senergy:
		return senergy.Factory
	case Mqtt:
		return mqtt.Factory
	default:
		panic("unknown connector type")
	}
}

func GetConnectorType(str string) (ConnectorType, error) {
	switch str {
	case "SENERGY":
		return Senergy, nil
	case "MQTT":
		return Mqtt, nil
	default:
		return 0, errors.New("unknown connector: " + str)
	}
}
