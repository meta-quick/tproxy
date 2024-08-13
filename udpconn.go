package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/juju/ratelimit"
	"github.com/meta-quick/gocodec"
	"github.com/meta-quick/tproxy/display"
	"github.com/meta-quick/tproxy/protocol"
)

type UdpPairedConnection struct {
	id       int
	cliAddr  net.UDPAddr
	cliConn  *UdpClient
	svrConn  net.Conn
	once     sync.Once
	stopChan chan struct{}
}

func NewUdpPairedConnection(id int, svrConn net.Conn, cliAddr net.UDPAddr) *UdpPairedConnection {
	cliConn := NewUdpClient(id, cliAddr)
	return &UdpPairedConnection{
		id:       id,
		cliAddr:  cliAddr,
		svrConn:  svrConn,
		cliConn:  cliConn,
		stopChan: make(chan struct{}),
	}
}

func (c *UdpPairedConnection) Write(buffer []byte) (n int, err error) {
	return c.cliConn.buffer.Write(buffer)
}

func (c *UdpPairedConnection) copyData(dst io.Writer, src io.Reader, tag string) {
	_, e := io.Copy(dst, src)
	if e != nil {
		netOpError, ok := e.(*net.OpError)
		if ok && netOpError.Err.Error() != useOfClosedConn {
			reason := netOpError.Unwrap().Error()
			display.PrintlnWithTime(color.HiRedString("[%d] %s error, %s", c.id, tag, reason))
		}
	}
}

func (c *UdpPairedConnection) copyDataWithRateLimit(dst io.Writer, src io.Reader, tag string, limit int64) {
	if limit > 0 {
		bucket := ratelimit.NewBucket(time.Second, limit)
		src = ratelimit.Reader(src, bucket)
	}

	c.copyData(dst, src, tag)
}

func (c *UdpPairedConnection) handleClientMessage() {
	// client closed also trigger server close.
	defer c.stop()

	r, w := io.Pipe()
	tee := io.MultiWriter(c.svrConn, w)
	go protocol.CreateInterop(settings.Protocol).Dump(r, protocol.ClientSide, c.id, settings.Quiet)
	c.copyDataWithRateLimit(tee, c.cliConn, protocol.ClientSide, settings.UpLimit)
}

func (c *UdpPairedConnection) handleServerMessage() {
	// server closed also trigger client close.
	defer c.stop()

	r, w := io.Pipe()
	tee := io.MultiWriter(newDelayedWriter(c.cliConn, settings.Delay, c.stopChan), w)
	go protocol.CreateInterop(settings.Protocol).Dump(r, protocol.ServerSide, c.id, settings.Quiet)
	c.copyDataWithRateLimit(tee, c.svrConn, protocol.ServerSide, settings.DownLimit)
}

func (c *UdpPairedConnection) process() {
	defer c.stop()

	conn, err := net.Dial("udp", settings.Remote)
	if err != nil {
		display.PrintlnWithTime(color.HiRedString("[x][%d] Couldn't connect to server: %v", c.id, err))
		return
	}

	display.PrintlnWithTime(color.HiGreenString("[%d] Connected to server: %s", c.id, conn.RemoteAddr()))

	c.svrConn = conn
	go c.handleServerMessage()

	c.handleClientMessage()
}

func (c *UdpPairedConnection) stop() {
	c.once.Do(func() {
		close(c.stopChan)

		if c.cliConn != nil {
			display.PrintlnWithTime(color.HiBlueString("[%d] Client connection closed", c.id))
		}
		if c.svrConn != nil {
			display.PrintlnWithTime(color.HiBlueString("[%d] Server connection closed", c.id))
			c.svrConn.Close()
		}
	})
}

func UdpRelayListener() error {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(settings.LocalHost),
		Port: settings.LocalPort,
	})

	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer conn.Close()

	display.PrintfWithTime("Listening on %s...\n", conn.LocalAddr().String())

	var connIndex int
	buf := make([]byte, 8192)
	udpmap := make(map[string]*UdpPairedConnection)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf("server: accept: %w", err)
		}

		connIndex++
		display.PrintlnWithTime(color.HiGreenString("[%d] Packet from: %s",
			connIndex, addr))
		if n <= 0 {
			continue
		}
		pconn, ok := udpmap[addr.String()]
		if !ok {
			pconn = NewUdpPairedConnection(connIndex, conn, *addr)
			udpmap[addr.String()] = pconn
		}

		//put data
		pconn.Write(buf[:n])
		go pconn.process()
	}
}

type UdpClient struct {
	conn    *net.UDPConn
	id      int
	cliAddr net.UDPAddr
	buffer  gocodec.Buffer
}

func NewUdpClient(id int, cliAddr net.UDPAddr) *UdpClient {
	t := &UdpClient{id: id, cliAddr: cliAddr, buffer: gocodec.Buffer{}}
	return t
}

func (c *UdpClient) Read(b []byte) (n int, err error) {
	return c.buffer.ReadLess(b)
}

func (c *UdpClient) Write(b []byte) (n int, err error) {
	return c.conn.WriteToUDP(b, &c.cliAddr)
}

type UdpRelay struct {
}

func NewUdpRelay() *UdpRelay {
	t := &UdpRelay{}
	return t
}

func (t *UdpRelay) StartListener() error {
	err := UdpRelayListener()
	return err
}
