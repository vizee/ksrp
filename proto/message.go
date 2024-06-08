package proto

import (
	"io"
	"net"
	"time"
)

const (
	CmdError        = 0
	CmdShakeHands   = 0x7d
	CmdShakeHandsOk = 0x7e
)

func ReadMessage(conn net.Conn, timeout time.Duration) (byte, string, error) {
	var (
		header [2]byte
		token  [256]byte
	)
	if timeout > 0 {
		conn.SetReadDeadline(time.Now().Add(timeout))
	}
	_, err := io.ReadFull(conn, header[:])
	if err == nil {
		_, err = io.ReadFull(conn, token[:header[1]])
	}
	if timeout > 0 {
		conn.SetReadDeadline(time.Time{})
	}
	if err != nil {
		return 0, "", err
	}
	return header[0], string(token[:header[1]]), nil
}

func WriteMessage(conn net.Conn, cmd byte, msg string, timeout time.Duration) error {
	buf := make([]byte, 0, 2+len(msg))
	buf = append(buf, cmd, byte(len(msg)))
	buf = append(buf, msg...)
	if timeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(time.Second))
	}
	_, err := conn.Write(buf)
	if timeout > 0 {
		conn.SetWriteDeadline(time.Time{})
	}
	return err
}
