package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"crossocean.com/edge_gateway/pkg/mqtt"
	"crossocean.com/edge_gateway/pkg/udp"
)

const (
	udpAddr      = "127.0.0.2:8888"
	mqttBroker   = "tcp://139.196.203.208:1883"
	mqttUser     = "hzy"
	mqttPassword = "ipac_1234"
	mqttTopic    = "root/root_01/navigation/gps/v1.0/report"
)

func main() {
	// 创建数据通道
	dataChan := make(chan []byte, 100)

	// 创建并连接MQTT客户端
	mqttClient := mqtt.NewClient(mqtt.Config{
		Broker:   mqttBroker,
		User:     mqttUser,
		Password: mqttPassword,
	})

	if err := mqttClient.Connect(); err != nil {
		log.Fatalf("Failed to connect MQTT: %v", err)
	}
	defer mqttClient.Disconnect(250)

	// 启动MQTT发布协程
	mqttClient.StartPublisher(mqttTopic, dataChan)

	// 创建并启动UDP监听器
	udpListener := udp.NewListener(udpAddr, dataChan)
	if err := udpListener.Start(); err != nil {
		log.Fatalf("Failed to start UDP listener: %v", err)
	}
	defer udpListener.Close()

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down service...")
	close(dataChan)
	time.Sleep(time.Second)
}
