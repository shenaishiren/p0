// Implementation of a MultiEchoServer. Students should write their code in this file.

package p0

import (
	"bufio"
	"strconv"
	"net"
	"fmt"
)

type multiEchoServer struct {
	ln               net.Listener
	clients          map[string]*multiEchoClient
	addClientChan    chan *multiEchoClient
	removeClientChan chan *multiEchoClient
	readChan         chan []byte
	closeChan        chan bool
	countRequestChan chan chan int
}

type multiEchoClient struct {
	conn      net.Conn // The client's TCP connection.
	writeChan chan []byte
	closeChan chan bool
}

// New creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {
	mes := new(multiEchoServer)
	mes.clients = make(map[string]*multiEchoClient)
	mes.addClientChan = make(chan *multiEchoClient)
	mes.removeClientChan = make(chan *multiEchoClient)
	mes.readChan = make(chan []byte)
	mes.closeChan = make(chan bool)
	mes.countRequestChan = make(chan chan int)
	return mes
}

func (mes *multiEchoServer) Start(port int) error {
	fmt.Println("Start listening...")
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	mes.ln = listener
	//调用服务例程
	go mes.handleAccept(mes.ln)
	go mes.serverRoutine()
	return err
}

func (mes *multiEchoServer) Close() {
	mes.closeChan <- true
	mes.ln.Close()
}

func (mes *multiEchoServer) Count() int {
	countChan := make(chan int)
	mes.countRequestChan <- countChan
	return <- countChan
}

func (mes *multiEchoServer) serverRoutine() {
	// Event loop
	for {
		select {
		// 从客户端读数据
		case line := <-mes.readChan:
			for _, client := range mes.clients {
				if len(client.writeChan) < 100 {
					client.writeChan <- line
				}
			}
		// 连接客户端
	    case client := <-mes.addClientChan:
			mes.clients[client.conn.RemoteAddr().String()] = client
			go client.clientRoutine(mes)
			go mes.handleRead(client)
		// 删除一个客户端的连接
		case client := <-mes.removeClientChan:
			client.closeChan <- true
			delete(mes.clients, client.conn.RemoteAddr().String())
		// 连接数请求
		case countChan := <-mes.countRequestChan:
			countChan <- len(mes.clients)
		// 关闭所有连接
		case <- mes.closeChan:
			for _, client := range mes.clients {
				client.closeChan <- true
				delete(mes.clients, client.conn.RemoteAddr().String())
			}
			return
		}
	}
}

func (client *multiEchoClient) clientRoutine(mes *multiEchoServer) {
	// Event loop
	for {
		select {
		// 给客户端写数据
	    case line := <-client.writeChan:
			_, err := client.conn.Write(line)
			if err != nil {
				mes.removeClientChan <- client
				return
			}
		// 关闭客户端连接
		case <-client.closeChan:
			client.conn.Close()
			return
		}
	}
}

func (mes *multiEchoServer) handleAccept(ln net.Listener)  {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		client := new(multiEchoClient)
		client.conn = conn
		client.writeChan = make(chan []byte, 100)
		client.closeChan = make(chan bool)
		mes.addClientChan <- client
	}
}

func (mes *multiEchoServer) handleRead(client *multiEchoClient) {
	reader := bufio.NewReader(client.conn)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			mes.removeClientChan <- client
			return
		}
		mes.readChan <- line
	}
}

// TODO: add additional methods/functions below!
