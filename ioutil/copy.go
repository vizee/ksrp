package ioutil

import (
	"io"
	"sync"
	"sync/atomic"
)

func DualCopy(cc io.ReadWriter, sc io.ReadWriter) error {
	const copyBufferSize = 16 * 1024

	var (
		once  atomic.Bool
		ioerr error
		sema  sync.WaitGroup
	)

	copy := func(dst io.ReadWriter, src io.ReadWriter) {
		var err error
		if rd, ok := dst.(io.ReaderFrom); ok {
			// 避免 io.CopyBuffer 在 WriteTo 和 ReadFrom 调用后丢失 buffer
			_, err = rd.ReadFrom(src)
		} else {
			_, err = io.CopyBuffer(dst, src, make([]byte, copyBufferSize))
		}
		if once.CompareAndSwap(false, true) {
			ioerr = err
			sema.Done()
		}
	}

	sema.Add(1)

	go copy(cc, sc)
	go copy(sc, cc)

	sema.Wait()

	return ioerr
}
