package Cell

import (
	"bufio"
	"fmt"
	"net"
)

type CellAdvisor struct {
	ip     string
	reader *bufio.Reader
	writer *bufio.Writer
}

func (self CellAdvisor) SendMessage(cmd byte, data string) {

	sending_msg := ""
	sending_msg = string([]byte{0x7f, 'C', cmd, 0x01, 0x01})
	if data != "" {
		sending_msg += data
	}
	sending_msg += string(self.getChecksum(sending_msg[1:]))
	sending_msg += string([]byte{0x7e})

	fmt.Fprintf(self.writer, string(sending_msg))
	self.writer.Flush()
}

func (self CellAdvisor) GetMessage() []byte {

	is_marked := false
	result, buf_result := []byte{}, []byte{}
	message_continue := true
	for message_continue {
		ret, err := self.reader.ReadBytes(0x7e)
		if err != nil {
			panic(err)
		}
		//delete subsequent checksum if its marked
		if ret[len(ret)-2] == '}' {
			ret = ret[:len(ret)-3]
		} else {
			ret = ret[:len(ret)-2]
		}
		buf_result = append(buf_result, ret[5:]...)
		if ret[3] <= ret[4]+1 {
			message_continue = false
		}
	}

buffer_loop:
	for _, value := range buf_result {
		if value == '}' {
			is_marked = true
			continue buffer_loop
		} else if is_marked {
			switch value {
			case 93, 94, 95:
				value = value ^ 0x20
			}
			is_marked = false
		}
		result = append(result, byte(value))
	}
	return result
}

func (self *CellAdvisor) initCellAdvisor() {

	conn, err := net.Dial("tcp", self.ip+":66")
	if err != nil {
		panic(err)
	}

	self.reader, self.writer = bufio.NewReader(conn), bufio.NewWriter(conn)
}

func (self CellAdvisor) GetScreen() []byte {
	self.SendMessage(0x60, "")
	return self.GetMessage()
}

func (self CellAdvisor) SendSCPI(scpicmd string) {
	self.SendMessage(0x61, scpicmd+"\n")
}

func (self CellAdvisor) getChecksum(data string) []byte {
	total := 0
	for _, value := range data {
		total += int(value)
	}
	buff := total & 0xff
	switch buff {
	case 0x7e, 0x7d, 0x7f:
		return []byte{0x7d, byte(buff ^ 0x20)}
	}
	return []byte{byte(buff)}
}

func NewCellAdvisor(ip string) CellAdvisor {
	cell := CellAdvisor{ip: ip}
	cell.initCellAdvisor()
	return cell
}
