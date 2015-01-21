package cell

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"
)

func randomBytes(l int) []byte {
	rand.Seed(time.Now().UTC().UnixNano())
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(0, 255))
	}
	return bytes
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func buildFakeCellAdvisorTCPConnection() <-chan int {
	done := make(chan int)
	go func() {

		l, err := net.Listen("tcp", JDProtocolPort)
		if err != nil {
			log.Fatal(err)
		}
		close(done)
		defer l.Close()
		for {
			// Wait for a connection.
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}
			go func(c net.Conn) {
				reader := bufio.NewReader(c)
				for {
					buf, err := reader.ReadBytes(0x7e)
					if err != nil {
						fmt.Println("Error reading:", err.Error())
					}
					c.Write(buf)
				}
				c.Close()
			}(conn)
		}
	}()
	return done
}
func TestSendMessageAndGetMessageSync(t *testing.T) {
	var compare []byte
	done := buildFakeCellAdvisorTCPConnection()
	JDProtocolPort = ":18081"
	<-done
	cl := NewCellAdvisor("")
	for i := 0; i < 200; i++ {

		result := randomBytes(100)
		_, err := cl.SendMessage(0x50, string(result[:]))
		if err != nil {
			t.Fatal(err)
			return
		}

		compare = []byte("")
		compare, err = cl.GetMessage()
		if err != nil {
			t.Fatal(err)
			return
		}
		if !bytes.Equal(result, compare) {
			t.Fatal("\nsend=", result, "\nget=", compare, ", not same")
		}
	}
}
