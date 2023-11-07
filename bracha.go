package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

const f = 1
const N = 3*f + 1

// Message 结构体用于存储消息的类型和内容。
type Message struct {
	Type    string
	Content string
}

// Party 结构体代表一个参与者或节点，其中包括其状态、互斥锁、已接收的消息等。
type Party struct {
	id            int
	sentReady     bool
	mu            sync.Mutex
	receivedEcho  map[string]int
	receivedReady map[string]int
	isBad         bool
	bracha        *Bracha
	content       string
}

// Bracha 结构体代表整个系统，其中包含所有的参与者或节点。
type Bracha struct {
	parties []*Party
}

// NewBracha 函数初始化一个新的 Bracha 系统，其中包含 N 个节点。
func NewBracha() *Bracha {
	log.Println("Initializing new Bracha instance...")
	bracha := &Bracha{}
	for i := 0; i < N; i++ {
		node := &Party{
			isBad:         false,
			id:            i,
			receivedEcho:  make(map[string]int),
			receivedReady: make(map[string]int),
			bracha:        bracha,
			content:       "",
		}
		bracha.parties = append(bracha.parties, node)
		log.Printf("Added party with ID: %d", i)
	}
	return bracha
}

// ReceiveMessage 方法处理从其他节点接收到的消息，并根据需要更新当前节点的状态。
func (p *Party) ReceiveMessage(m Message) {
	log.Printf("Party%d received %s message: %s", p.id, m.Type, m.Content)
	p.mu.Lock()
	defer p.mu.Unlock()

	var broadcastMessage *Message

	switch m.Type {
	case "propose":
		if p.isBad {
			broadcastMessage = &Message{Type: "echo", Content: m.Content + fmt.Sprint(114514)}
		} else {
			broadcastMessage = &Message{Type: "echo", Content: m.Content}
		}

	case "echo":
		p.receivedEcho[m.Content]++
		if p.receivedEcho[m.Content] >= 2*f+1 && !p.sentReady {
			p.sentReady = true
			broadcastMessage = &Message{Type: "ready", Content: m.Content}
		}

	case "ready":
		p.receivedReady[m.Content]++
		if p.receivedReady[m.Content] >= 2*f+1 {
			p.content = m.Content
		}
	}

	if broadcastMessage != nil {
		log.Printf("Party%d broadcasting %s message: %s", p.id, broadcastMessage.Type, broadcastMessage.Content)
		p.Broadcast(*broadcastMessage)
	}
}

// Broadcast 方法向所有其他节点广播消息。
func (p *Party) Broadcast(m Message) {
	for _, party := range p.bracha.parties {
		go party.ReceiveMessage(m)
	}
}

func main() {
	// 创建一个新的 Bracha 实例。
	brachaInstance := NewBracha()

	// 将第三个 Party 设置为恶意节点。
	brachaInstance.parties[2].isBad = true
	log.Println("Party2 has been set as malicious.")

	// 从第一个 Party 广播一个提议消息。
	brachaInstance.parties[0].Broadcast(Message{Type: "propose", Content: "ApexQIDONG!"})

	// 等待所有消息传递完毕
	time.Sleep(1 * time.Second)

	// 打印每个 Party 的内容。
	for _, party := range brachaInstance.parties {
		fmt.Printf("Node%d says: %s\n", party.id, party.content)
	}
}
