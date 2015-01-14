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
func (cl CellAdvisor) SendMessage(cmd byte, data string) (int, error) {

	sendingMsg := ""
	sendingMsg = string([]byte{0x7f, 'C', cmd, 0x01, 0x01})
	if data != "" {
		sendingMsg += data
	}
	sendingMsg += string(cl.getChecksum(sendingMsg[1:]))
	sendingMsg += string([]byte{0x7e})

	num, err := fmt.Fprintf(cl.writer, string(sendingMsg))
	if err != nil {
		return num, err
	}
	err = cl.writer.Flush()
	if err != nil {
		return num, err
	}
	return num, err
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
func (cl CellAdvisor) SendSCPI(scpicmd string) (int, error) {
	return cl.SendMessage(0x61, scpicmd+"\n")
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
