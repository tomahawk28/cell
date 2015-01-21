//JDSU CellAdvisor Go Library
//Copyright (C) 2015 Jihyuk Bok <tomahawk28@gmail.com>
//
//Permission is hereby granted, free of charge, to any person obtaining
//a copy of this software and associated documentation files (the "Software"),
//to deal in the Software without restriction, including without limitation
//the rights to use, copy, modify, merge, publish, distribute, sublicense,
//and/or sell copies of the Software, and to permit persons to whom the
//Software is furnished to do so, subject to the following conditions:
//
//The above copyright notice and this permission notice shall be included
//in all copies or substantial portions of the Software.
//
//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
//EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
//OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
//IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
//DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
//TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
//OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package cell

import (
	"bufio"
	"log"
	"net"
)

var (
	JDProtocolPort = ":66"
)

// CellAdvisor represents connection status with JDSU CellAdvisor devices
// It could only made by NewCellAdvisor(ip string) function
type CellAdvisor struct {
	ip     string
	reader *bufio.Reader
	writer *bufio.Writer
}

// SendMessage could send single cmd byte, and data strings
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

func (cl *CellAdvisor) initCellAdvisor() {

	conn, err := net.Dial("tcp", cl.ip+JDProtocolPort)
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
	var masking bool = false
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
