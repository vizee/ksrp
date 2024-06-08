//go:build unix

package main

import (
	"io"
	"net"
	"syscall"
	"unsafe"
)

func sysrecv(fd uintptr, buf []byte, flags int) (uintptr, syscall.Errno) {
	// 没有阻塞，直接用 RawSyscall
	n, _, err := syscall.RawSyscall6(syscall.SYS_RECVFROM,
		uintptr(fd),
		uintptr(unsafe.Pointer(unsafe.SliceData(buf))),
		uintptr(len(buf)),
		uintptr(flags),
		0, 0)
	return n, err
}

// checkTcpRead 返回 nil 表示连接有数据，EOF 表示对端关闭
func checkTcpRead(conn *net.TCPConn) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	var innerErr error
	err = rawConn.Read(func(fd uintptr) bool {
		var buf [1]byte
		n, eno := sysrecv(fd, buf[:], syscall.MSG_PEEK)
		if eno != 0 {
			if eno == syscall.EAGAIN {
				// 阻塞等待读就绪
				return false
			}
			innerErr = eno
		} else if n == 0 {
			// 对端关闭连接
			innerErr = io.EOF
		}
		return true
	})
	if err != nil {
		return err
	}

	return innerErr
}
