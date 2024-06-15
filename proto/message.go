package proto

import (
	"io"
)

const (
	CmdError        = 0
	CmdShakeHands   = 0x7d
	CmdShakeHandsOk = 0x7e
)

func ReadMessage(conn io.Reader) (byte, string, error) {
	var (
		header [2]byte
		token  [256]byte
	)
	_, err := io.ReadFull(conn, header[:])
	if err == nil {
		_, err = io.ReadFull(conn, token[:header[1]])
	}
	if err != nil {
		return 0, "", err
	}
	return header[0], string(token[:header[1]]), nil
}

func WriteMessage(conn io.Writer, cmd byte, msg string) error {
	buf := make([]byte, 0, 2+len(msg))
	buf = append(buf, cmd, byte(len(msg)))
	buf = append(buf, msg...)
	_, err := conn.Write(buf)
	return err
}
