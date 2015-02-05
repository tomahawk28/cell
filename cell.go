// Package cell provides api connections between JDSU CellAdvisor Devices
package cell

import (
	"bufio"
	"errors"
	"log"
	"net"
	"regexp"
	"strconv"
)

var (
	//JDProtocolPort represents Port number which CellAdviosr TCP Connection uses
	JDProtocolPort = ":66"
)

// CellAdvisor represents connection status with JDSU CellAdvisor devices
// It could only made by NewCellAdvisor(ip string) function
type CellAdvisor struct {
	ip     string
	reader *bufio.Reader
	writer *bufio.Writer
}

// InterferencePower represents interferences data given by CellAdvisor
type InterferencePower struct {
	// Unit represents real unit of following float array
	Unit string `json:"Unit"`
	// PowerTrace is measured power array based on frequency plan
	Powertrace []float32 `json:"Powertrace"`
}

// SendMessage send single cmd byte, and data strings and returing
// the number of bytes send and followed error if any
func (cl CellAdvisor) SendMessage(cmd byte, data string) (int, error) {

	sendingMsg := []byte{'C', cmd, 0x01, 0x01}
	if data != "" {
		sendingMsg = append(sendingMsg, []byte(data)...)
	}
	//append checksum
	sendingMsg = append(sendingMsg, getChecksum(sendingMsg))
	sendingMsg = maskingCommandCharacter(sendingMsg)
	sendingMsg = append(append([]byte{0x7f}, sendingMsg...), 0x7e)

	num, err := cl.writer.Write(sendingMsg)
	//num, err := fmt.Fprintf(cl.writer, string(sendingMsg))
	err = cl.writer.Flush()
	return num, err
}

// GetMessage receive data right after it send request with SendMessage
// it returns the data and followed error if any
func (cl CellAdvisor) GetMessage() ([]byte, error) {

	bufResult := []byte{}
	messageContinue := true
	for messageContinue {
		ret, err := cl.reader.ReadBytes(0x7e)
		if err != nil {
			return nil, err
		}
		command, checksum := unmaskingCommandCharacter(ret[5 : len(ret)-1])
		if realchecksum := getChecksum(append(ret[1:5], command...)); realchecksum != checksum {
			log.Printf("checksum required to be: %c, but %c", realchecksum, checksum)
		}
		bufResult = append(bufResult, command...)
		if ret[3] <= ret[4]+1 {
			messageContinue = false
		}
	}

	return bufResult, nil
}

func (cl *CellAdvisor) Reinitialize() {
	cl.initCellAdvisor()
}

func (cl *CellAdvisor) initCellAdvisor() {

	conn, err := net.Dial("tcp", cl.ip+JDProtocolPort)
	if err != nil {
		panic(err)
	}

	cl.reader, cl.writer = bufio.NewReader(conn), bufio.NewWriter(conn)
}

// GetScreen returning current devices jpeg screenshot
func (cl CellAdvisor) GetScreen() ([]byte, error) {
	cl.SendMessage(0x60, "")
	return cl.GetMessage()
}

// GetStatusMessage returning a heartbeat signal message from CellAdvisor
func (cl CellAdvisor) GetStatusMessage() (string, error) {
	cl.SendMessage(0x50, "")
	ret, err := cl.GetMessage()
	return string(ret), err
}

// GetInterferencePower returning current interference power array
// with json format
func (cl CellAdvisor) GetInterferencePower() (*InterferencePower, error) {
	cl.SendMessage(0x83, "")
	ret, err := cl.GetMessage()
	if err != nil {
		return nil, err
	}

	reunit := regexp.MustCompile("Unit=\"([a-zA-Z]+)\"")
	repower := regexp.MustCompile("P([0-9]+)+=\"(-*[0-9]+.[0-9]+)\"")

	unit := reunit.FindStringSubmatch(string(ret))
	trace := repower.FindAllStringSubmatch(string(ret), -1)
	if trace == nil || unit == nil {
		return nil, errors.New("input is not an interference XML source")
	}

	powertrace := make([]float32, len(trace))
	for i, v := range trace {
		convertfloatResult, err := strconv.ParseFloat(v[len(v)-1], 32)
		powertrace[i] = float32(convertfloatResult)
		if err != nil {
			return nil, err
		}
	}
	return &InterferencePower{Unit: unit[1], Powertrace: powertrace}, nil
}

// SendSCPI sends SCPI commands to CellAdvisor devices
// (http://en.wikipedia.org/wiki/Standard_Commands_for_Programmable_Instruments)
func (cl CellAdvisor) SendSCPI(scpicmd string) (int, error) {
	return cl.SendMessage(0x61, scpicmd+"\n")
}

func maskingCommandCharacter(data []byte) []byte {
	var result []byte
	for _, character := range data {
		switch character {
		case 0x7e, 0x7d, 0x7f:
			result = append(result, 0x7d, 0x20^character)
		default:
			result = append(result, character)
		}
	}
	return result
}

func unmaskingCommandCharacter(data []byte) ([]byte, byte) {
	var result []byte
	var masking bool
	for _, character := range data {
		switch character {
		case 0x7d:
			masking = true
		default:
			if masking {
				masking = false
				result = append(result, character^0x20)
			} else {
				result = append(result, character)
			}
		}
	}
	return result[:len(result)-1], result[len(result)-1]
}

func getChecksum(data []byte) byte {
	total := 0
	for _, value := range data {
		total += int(value)
	}
	return byte(total & 0xff)
}

// NewCellAdvisor creates new CellAdvior object with given ip address
func NewCellAdvisor(ip string) CellAdvisor {
	cell := CellAdvisor{ip: ip}
	cell.initCellAdvisor()
	return cell
}
