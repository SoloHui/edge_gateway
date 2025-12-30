package mqtt

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client MQTT客户端
type Client struct {
	broker   string
	user     string
	password string
	clientID string
	client   mqtt.Client
}

// Config MQTT配置
type Config struct {
	Broker   string
	User     string
	Password string
	ClientID string
}

// NewClient 创建新的MQTT客户端
func NewClient(config Config) *Client {
	return &Client{
		broker:   config.Broker,
		user:     config.User,
		password: config.Password,
		clientID: config.ClientID,
	}
}

// Connect 连接到MQTT broker
func (c *Client) Connect() error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(c.broker)
	opts.SetUsername(c.user)
	opts.SetPassword(c.password)

	if c.clientID == "" {
		c.clientID = "edge_gateway_" + fmt.Sprintf("%d", time.Now().Unix())
	}
	opts.SetClientID(c.clientID)

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v\n", err)
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("MQTT connected successfully")
	})
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	c.client = mqtt.NewClient(opts)
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

// Publish 发布消息到指定主题
func (c *Client) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	token := c.client.Publish(topic, qos, retained, payload)
	token.Wait()
	return token.Error()
}

// StartPublisher 启动发布器，从通道读取数据并发布到MQTT
func (c *Client) StartPublisher(topic string, dataChan <-chan []byte) {
	go func() {
		for data := range dataChan {
			if err := c.Publish(topic, 0, false, data); err != nil {
				log.Printf("Failed to publish MQTT message: %v", err)
			}
		}
	}()
}

// Disconnect 断开MQTT连接
func (c *Client) Disconnect(quiesce uint) {
	log.Println("Disconnecting MQTT...")
	if c.client != nil {
		c.client.Disconnect(quiesce)
	}
}
