package mqtt

import (
	"encoding/json"
	"errors"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
)

func (this *Client) startMqtt() error {
	options := paho.NewClientOptions().
		SetPassword(this.password).
		SetUsername(this.userName).
		SetClientID(this.mqttClientId).
		SetAutoReconnect(true).
		SetCleanSession(true).
		AddBroker(this.mqttUrl).
		SetConnectionLostHandler(func(client paho.Client, err error) {
			log.Println("mqtt connection lost:", err)
		}).
		SetOnConnectHandler(func(client paho.Client) {
			log.Println("mqtt (re)connected")
			err := this.loadOldSubscriptions()
			if err != nil {
				log.Fatal("FATAL: ", err)
			}
		})
	this.mqtt = paho.NewClient(options)
	if token := this.mqtt.Connect(); token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Connect(): ", token.Error())
		return token.Error()
	}
	return nil
}

type Subscription struct {
	Topic   string
	Handler paho.MessageHandler
	Qos     byte
}

func (this *Client) registerSubscription(topic string, qos byte, handler paho.MessageHandler) {
	this.subscriptionsMux.Lock()
	defer this.subscriptionsMux.Unlock()
	this.subscriptions[topic] = Subscription{
		Topic:   topic,
		Handler: handler,
		Qos:     qos,
	}
}

func (this *Client) unregisterSubscriptions(topic string) {
	this.subscriptionsMux.Lock()
	defer this.subscriptionsMux.Unlock()
	delete(this.subscriptions, topic)
}

func (this *Client) getSubscriptions() (result []Subscription) {
	this.subscriptionsMux.Lock()
	defer this.subscriptionsMux.Unlock()
	for _, sub := range this.subscriptions {
		result = append(result, sub)
	}
	return
}

func (this *Client) loadOldSubscriptions() error {
	if !this.mqtt.IsConnected() {
		log.Println("WARNING: mqtt client not connected")
		return errors.New("mqtt client not connected")
	}
	subs := this.getSubscriptions()
	for _, sub := range subs {
		log.Println("resubscribe to", sub.Topic)
		token := this.mqtt.Subscribe(sub.Topic, sub.Qos, sub.Handler)
		if token.Wait() && token.Error() != nil {
			log.Println("Error on Subscribe: ", sub.Topic, token.Error())
			return token.Error()
		}
	}
	return nil
}

func (this *Client) PublishJson(topic string, msg interface{}, qos byte) (err error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return this.PublishStr(topic, string(payload), qos)
}

func (this *Client) PublishStr(topic string, msg string, qos byte) (err error) {
	if !this.mqtt.IsConnected() {
		log.Println("WARNING: mqtt client not connected")
		return errors.New("mqtt client not connected")
	}
	token := this.mqtt.Publish(topic, qos, false, msg)
	if token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Publish(): ", token.Error())
		return token.Error()
	}
	return err
}
