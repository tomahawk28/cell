package cell

import (
	"bufio"
	"fmt"
	"net"
)

// CellAdvisor represents connection status with JDSU CellAdvisor devices
// It could only made by NewCellAdvisor(ip string) function
type CellAdvisor struct {
	ip     string
	reader *bufio.Reader
	writer *bufio.Writer
}

// SendMessage could send single cmd byte, and data strings
func (cl CellAdvisor) SendMessage(cmd byte, data string) {

	sendingMsg := ""
	sendingMsg = string([]byte{0x7f, 'C', cmd, 0x01, 0x01})
	if data != "" {
		sendingMsg += data
	}
	sendingMsg += string(cl.getChecksum(sendingMsg[1:]))
	sendingMsg += string([]byte{0x7e})

	fmt.Fprintf(cl.writer, string(sendingMsg))
	cl.writer.Flush()
}

func (cl CellAdvisor) GetMessage() ([]byte, error) {

	isMarked := false
	result, bufResult := []byte{}, []byte{}
	messageContinue := true
	for messageContinue {
		ret, err := cl.reader.ReadBytes(0x7e)
		if err != nil {
			return nil, err
		}
		//delete subsequent checksum if its marked
		if ret[len(ret)-2] == '}' {
			ret = ret[:len(ret)-3]
		} else {
			ret = ret[:len(ret)-2]
		}
		bufResult = append(bufResult, ret[5:]...)
		if ret[3] <= ret[4]+1 {
			messageContinue = false
		}
	}

buffer_loop:
	for _, value := range bufResult {
		if value == '}' {
			isMarked = true
			continue buffer_loop
		} else if isMarked {
			switch value {
			case 93, 94, 95:
				value = value ^ 0x20
			}
			isMarked = false
		}
		result = append(result, byte(value))
	}
	return result, nil
}

func (cl *CellAdvisor) initCellAdvisor() {

	conn, err := net.Dial("tcp", cl.ip+":66")
	if err != nil {
		panic(err)
	}

	cl.reader, cl.writer = bufio.NewReader(conn), bufio.NewWriter(conn)
}

func (cl CellAdvisor) GetScreen() ([]byte, error) {
	cl.SendMessage(0x60, "")
	return cl.GetMessage()
}

func (cl CellAdvisor) GetStatusMessage() ([]byte, error) {
	cl.SendMessage(0x50, "")
	return cl.GetMessage()
}

// SendSCPI sends SCPI commands to CellAdvisor devices
// (http://en.wikipedia.org/wiki/Standard_Commands_for_Programmable_Instruments)
func (cl CellAdvisor) SendSCPI(scpicmd string) {
	cl.SendMessage(0x61, scpicmd+"\n")
}

func (cl CellAdvisor) getChecksum(data string) []byte {
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

// NewCellAdvisor creates new CellAdvior object with given ip address
func NewCellAdvisor(ip string) CellAdvisor {
	cell := CellAdvisor{ip: ip}
	cell.initCellAdvisor()
	return cell
}
