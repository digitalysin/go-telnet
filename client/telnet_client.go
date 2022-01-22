package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const defaultBufferSize = 4096

// TelnetClient represents a TCP client which is responsible for writing input data and printing response.
type TelnetClient struct {
	destination     *net.TCPAddr
	responseTimeout time.Duration
	dialTimeout     time.Duration
	connection      net.Conn
}

// NewTelnetClient method creates new instance of TCP client.
func NewTelnetClient(options Options) (*TelnetClient, error) {
	tcpAddr := createTCPAddr(options)
	resolved := resolveTCPAddr(tcpAddr)

	var (
		addr      = fmt.Sprintf("%s:%d", resolved.IP, resolved.Port)
		conn, err = net.DialTimeout("tcp", addr, options.DialTimeout())
	)

	if err != nil {
		return nil, err
	}

	return &TelnetClient{
		destination:     resolved,
		responseTimeout: options.Timeout(),
		dialTimeout:     options.DialTimeout(),
		connection:      conn,
	}, nil
}

func createTCPAddr(options Options) string {
	var buffer bytes.Buffer
	buffer.WriteString(options.Host())
	buffer.WriteByte(':')
	buffer.WriteString(fmt.Sprintf("%d", options.Port()))
	return buffer.String()
}

func resolveTCPAddr(addr string) *net.TCPAddr {
	resolved, error := net.ResolveTCPAddr("tcp", addr)
	if nil != error {
		log.Fatalf("Error occured while resolving TCP address \"%v\": %v\n", addr, error)
	}

	return resolved
}

func (t *TelnetClient) Close() error {
	return t.connection.Close()
}

// ProcessData method processes data: reads from input and writes to output.
func (t *TelnetClient) ProcessData(inputData io.Reader, outputData io.Writer) error {
	var (
		connection = t.connection
	)

	requestDataChannel := make(chan []byte)
	doneChannel := make(chan bool)
	responseDataChannel := make(chan []byte)

	go t.readInputData(inputData, requestDataChannel, doneChannel)
	go t.readServerData(connection, responseDataChannel)

	var afterEOFResponseTicker = new(time.Ticker)
	var afterEOFMode bool
	var somethingRead bool

	for {
		select {
		case request := <-requestDataChannel:
			if _, error := connection.Write(request); nil != error {
				return fmt.Errorf("Error occured while writing to TCP socket: %v\n", error)
			}
		case <-doneChannel:
			afterEOFMode = true
			afterEOFResponseTicker = time.NewTicker(t.responseTimeout)
		case response := <-responseDataChannel:
			outputData.Write([]byte(fmt.Sprintf("%v", string(response))))
			somethingRead = true

			if afterEOFMode {
				afterEOFResponseTicker.Stop()
				afterEOFResponseTicker = time.NewTicker(t.responseTimeout)
			}
		case <-afterEOFResponseTicker.C:
			if !somethingRead {
				return fmt.Errorf("Nothing read. Maybe connection timeout.")
			}
			return nil
		}
	}
}

func (t *TelnetClient) readInputData(inputData io.Reader, toSent chan<- []byte, doneChannel chan<- bool) {
	buffer := make([]byte, defaultBufferSize)
	var error error
	var n int

	reader := bufio.NewReader(inputData)

	for nil == error {
		n, error = reader.Read(buffer)
		toSent <- buffer[:n]
	}

	t.assertEOF(error)
	doneChannel <- true
}

func (t *TelnetClient) readServerData(connection net.Conn, received chan<- []byte) {
	buffer := make([]byte, defaultBufferSize)
	var error error
	var n int

	for nil == error {
		n, error = connection.Read(buffer)
		received <- buffer[:n]
	}

	t.assertEOF(error)
}

func (t *TelnetClient) assertEOF(error error) {
	if "EOF" != error.Error() {
		log.Fatalf("Error occured while operating on TCP socket: %v\n", error)
	}
}
