package main

import (
	"log/slog"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type localConn struct {
	conn net.Conn
	idx  int
	free atomic.Bool
}

type localPool struct {
	address    string
	preconnect int
	avail      []*localConn
	lock       sync.Mutex
	cond       sync.Cond
}

func (p *localPool) checkConnAlive(c *localConn) {
	err := checkTcpRead(c.conn.(*net.TCPConn))
	// TCP 读出现错误或者提前有数据到达都认为是异常情况
	if err != nil || !c.free.Load() {
		slog.Warn("checkTcpRead", "conn", c.conn.RemoteAddr().String(), "free", c.free.Load(), "err", err)
		p.lock.Lock()
		ok := p.removeConnLocked(c)
		p.lock.Unlock()
		if ok {
			c.conn.Close()
		}
		return
	}
}

func (p *localPool) addConn() {
retry:
	conn, err := net.Dial("tcp", p.address)
	if err != nil {
		slog.Warn("dial", "address", p.address, "err", err)
		time.Sleep(time.Second)
		goto retry
	}
	c := &localConn{
		conn: conn,
	}
	p.lock.Lock()
	c.idx = len(p.avail)
	p.avail = append(p.avail, c)
	p.lock.Unlock()

	go p.checkConnAlive(c)
}

func (p *localPool) connect() {
	for {
		p.lock.Lock()
		for len(p.avail) >= p.preconnect {
			p.cond.Wait()
		}
		p.lock.Unlock()

		p.addConn()
	}
}

func (p *localPool) removeConnLocked(c *localConn) bool {
	if c.free.Load() {
		return false
	}
	c.free.Store(true)
	lastIdx := len(p.avail) - 1
	p.avail[lastIdx].idx = c.idx
	p.avail[c.idx] = p.avail[lastIdx]
	p.avail = p.avail[:lastIdx]

	if p.preconnect-lastIdx == 1 {
		p.cond.Signal()
	}
	return true
}

func (p *localPool) getAvail() (net.Conn, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.avail) == 0 {
		return nil, false
	}
	c := p.avail[rand.IntN(len(p.avail))]
	// cas 尽量获得有效的连接
	ok := p.removeConnLocked(c)
	if !ok {
		return nil, false
	}
	return c.conn, true
}

func (p *localPool) get() (net.Conn, error) {
	conn, ok := p.getAvail()
	if ok {
		return conn, nil
	}
	return net.Dial("tcp", p.address)
}

func startLocalPool(address string, preconnect int) *localPool {
	p := &localPool{
		address:    address,
		preconnect: preconnect,
	}
	p.cond.L = &p.lock

	if preconnect > 0 {
		go p.connect()
	}
	return p
}
