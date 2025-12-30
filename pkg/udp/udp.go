package udp

import (
	"log"
	"net"
)

const (
	bufferSize = 1024
)

// Listener UDP监听器
type Listener struct {
	addr     string
	dataChan chan<- []byte
	conn     *net.UDPConn
	done     chan struct{}
}

// NewListener 创建新的UDP监听器
func NewListener(addr string, dataChan chan<- []byte) *Listener {
	return &Listener{
		addr:     addr,
		dataChan: dataChan,
		done:     make(chan struct{}),
	}
}

// Start 启动UDP监听
func (l *Listener) Start() error {
	addr, err := net.ResolveUDPAddr("udp", l.addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	l.conn = conn
	log.Printf("Started listening on UDP port: %s\n", l.addr)

	go l.listen()
	return nil
}

// listen 监听UDP端口并将数据发送到通道
func (l *Listener) listen() {
	buffer := make([]byte, bufferSize)
	for {
		select {
		case <-l.done:
			return
		default:
		}

		n, _, err := l.conn.ReadFromUDP(buffer)
		if err != nil {
			// 检查是否是因为连接被关闭
			select {
			case <-l.done:
				return
			default:
				log.Printf("Failed to read UDP data: %v", err)
				continue
			}
		}

		if n > 0 {
			// 复制数据到新的切片
			data := make([]byte, n)
			copy(data, buffer[:n])

			// 非阻塞发送到通道
			select {
			case l.dataChan <- data:
			default:
				log.Println("Data channel full, dropping data")
			}
		}
	}
}

// Close 关闭UDP连接
func (l *Listener) Close() error {
	log.Println("Closing UDP listener...")
	close(l.done)
	if l.conn != nil {
		return l.conn.Close()
	}
	return nil
}
