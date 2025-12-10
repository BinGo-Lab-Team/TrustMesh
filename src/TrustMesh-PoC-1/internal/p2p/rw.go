package p2p

import (
	"TrustMesh-PoC-1/internal/db"
	"TrustMesh-PoC-1/internal/logger"
	"TrustMesh-PoC-1/internal/tools"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net"
	"sync/atomic"
	"time"

	"github.com/zeebo/blake3"
)

func lenReadUnit32(conn net.Conn, maxLength uint32, deadline time.Duration) ([]byte, error) {
	// 读取长度字段
	var lenBuf [4]byte
	if err := conn.SetReadDeadline(time.Now().Add(deadline)); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
		return nil, err
	}

	// 确认 Body 长度
	size := binary.BigEndian.Uint32(lenBuf[:])
	if size > math.MaxInt32 || size > maxLength {
		return nil, errors.New("length size too big")
	}

	// 读取 Body
	bodyBuf := make([]byte, int(size))
	if err := conn.SetReadDeadline(time.Now().Add(deadline)); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(conn, bodyBuf); err != nil {
		return nil, err
	}

	return bodyBuf, nil
}

// readLoop 读通道协程
func (c *Connection) readLoop() {
	defer c.OnceDone.Do(func() { close(c.Done) })

	ct := c.MainState.ConnectionTable

	// 握手阻塞
	if c.IsInitiator {
		ok := make(chan struct{})
		select {
		case <-c.Ready:
			close(ok)
		case <-tools.WaitTimeout(ok, 3*time.Second):
			return
		case <-c.Done:
			return
		}
	}

	// 读循环
	for {
		select {
		case <-c.Done:
			return
		default:
			// 读取包头
			var headerBuf [4]byte
			if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return
			}
			if _, err := io.ReadFull(c.Conn, headerBuf[:]); err != nil {
				return
			}
			protocolId := binary.BigEndian.Uint32(headerBuf[:])

			// 握手状态
			state := atomic.LoadInt32(&c.SessionState)

			logger.Test("Read header: %v", protocolId)

			// 握手
			if state != int32(StateCompleted) {
				if state == int32(StateWaitingInitial) && protocolId == MsgHandshakeHello {
					var bodyBuf [72]byte
					// 读取 Hello 消息
					if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
						return
					}
					if _, err := io.ReadFull(c.Conn, bodyBuf[:]); err != nil {
						return
					}

					// 处理消息
					writeBuf, remote, local, isPass := processingHandshakeHello(bodyBuf)
					c.RemoteHandshake = remote
					c.LocalHandshake = local

					// 变更状态
					if isPass {
						if err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
							return
						}
						if _, err := c.Conn.Write(writeBuf[:]); err != nil {
							return
						}
						if success := atomic.CompareAndSwapInt32(&c.SessionState, int32(StateWaitingInitial), int32(StateWaitingReply)); !success {
							return
						}
						continue
					} else {
						return
					}
				} else if state == int32(StateWaitingReply) && protocolId == MsgHandshakeConfirm {
					var bodyBuf [64]byte
					// 读取 Confirm 消息
					if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
						return
					}
					if _, err := io.ReadFull(c.Conn, bodyBuf[:]); err != nil {
						return
					}

					// 处理消息
					isPass := processingHandshakeConfirm(bodyBuf, c.RemoteHandshake, c.LocalHandshake)

					// 变更状态
					if isPass {
						if success := atomic.CompareAndSwapInt32(&c.SessionState, int32(StateWaitingReply), int32(StateCompleted)); !success {
							return
						}
						nodeId := blake3.Sum256(c.RemoteHandshake.PK[:])

						// 注册连接
						ct.Lock.Lock()
						v, isExist := ct.Connection[nodeId]
						if isExist {
							go ConnectionAlreadyExists(nodeId, v)
						}
						ct.Connection[nodeId] = c.IOC
						ct.Lock.Unlock()

						// 通知握手完成
						close(c.Ready)

						logger.Debug("Handshake done: %v", nodeId)

						// 通知连接完成
						select {
						case c.NodeId <- nodeId:
							continue
						default:
							continue
						}
					} else {
						return
					}
				} else {
					logger.Debug("Handshake failed")
					return
				}
			}

			// 业务路由
			switch protocolId {
			case MsgHeartbeat:
				var bodyBuf [4]byte
				if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
					return
				}
				if _, err := io.ReadFull(c.Conn, bodyBuf[:]); err != nil {
					return
				}
				if !processingHeartbeat(bodyBuf) {
					return
				}
			case MsgBootstrapReply:
				bodyBuf, err := lenReadUnit32(c.Conn, math.MaxInt32, 10*time.Second)
				if err != nil {
					return
				}
				go processingBootstrapReply(bodyBuf, db.GetDB())
			case MsgInquiryHaveProposal:
				var bodyBuf [72]byte
				if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
					return
				}
				if _, err := io.ReadFull(c.Conn, bodyBuf[:]); err != nil {
					return
				}
				go c.h.ProcessingInquiry(bodyBuf, c.MainState, c.IOC)
			case MsgProposalBody:
				bodyBuf, err := lenReadUnit32(c.Conn, math.MaxInt32, 10*time.Second)
				if err != nil {
					return
				}
				go c.h.ProcessingProposalBody(bodyBuf, c.MainState)
			case MsgInquiryReply:
				var bodyBuf [36]byte
				if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
					return
				}
				if _, err := io.ReadFull(c.Conn, bodyBuf[:]); err != nil {
					return
				}
				go c.h.ProcessingInquiryReply(bodyBuf, c.IOC)
			case MsgProposalSig:
				bodyBuf, err := lenReadUnit32(c.Conn, math.MaxInt32, 10*time.Second)
				if err != nil {
					return
				}
				go c.h.ProcessProposalSig(bodyBuf, c.MainState)
			default:
				logger.Debug("Unknow protocolId: %v", protocolId)
				return
			}
		}
	}
}

// writeLoop 写通道协程
func (c *Connection) writeLoop() {
	defer c.OnceDone.Do(func() { close(c.Done) })

	ct := c.MainState.ConnectionTable

	// 握手阻塞
	if !c.IsInitiator {
		ok := make(chan struct{})
		select {
		case <-c.Ready:
			close(ok)
		case <-tools.WaitTimeout(ok, 3*time.Second):
			return
		case <-c.Done:
			return
		}
	}

	// 写循环
	for {
		select {
		case <-c.Done:
			return
		default:
			// 握手状态
			state := atomic.LoadInt32(&c.SessionState)

			// 握手
			if state != int32(StateCompleted) {
				if state == int32(StateWaitingInitial) {
					// 准备 Hello 消息
					writeBuf, local, isPass := newHandshakeHello()
					c.LocalHandshake = local

					// 变更状态并发送消息
					if isPass {
						if err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
							return
						}
						if _, err := c.Conn.Write(writeBuf[:]); err != nil {
							return
						}
						if success := atomic.CompareAndSwapInt32(&c.SessionState, int32(StateWaitingInitial), int32(StateWaitingReply)); !success {
							return
						}
						continue
					} else {
						return
					}
				} else if state == int32(StateWaitingReply) {
					var headerBuf [4]byte
					if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
						return
					}
					if _, err := io.ReadFull(c.Conn, headerBuf[:]); err != nil {
						return
					}
					protocolId := binary.BigEndian.Uint32(headerBuf[:])

					// 判断回复的协议 ID 是否合法
					if protocolId != MsgHandshakeResponse {
						return
					}

					// 读取内容
					var bodyBuf [136]byte
					if err := c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
						return
					}
					if _, err := io.ReadFull(c.Conn, bodyBuf[:]); err != nil {
						return
					}

					// 处理消息
					writeBuf, remote, isPass := processingHandshakeResponse(bodyBuf, c.LocalHandshake)
					c.RemoteHandshake = remote

					if isPass {
						// 发送消息
						if err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
							return
						}
						if _, err := c.Conn.Write(writeBuf[:]); err != nil {
							return
						}
						if success := atomic.CompareAndSwapInt32(&c.SessionState, int32(StateWaitingReply), int32(StateCompleted)); !success {
							return
						}
						nodeId := blake3.Sum256(c.RemoteHandshake.PK[:])

						// 注册连接
						ct.Lock.Lock()
						v, isExist := ct.Connection[nodeId]
						if isExist {
							go ConnectionAlreadyExists(nodeId, v)
						}
						ct.Connection[nodeId] = c.IOC
						ct.Lock.Unlock()

						// 通知握手完成
						close(c.Ready)

						logger.Debug("Handshake done: %v", nodeId)

						// 通知连接完成
						select {
						case c.NodeId <- nodeId:
							continue
						default:
							continue
						}
					} else {
						return
					}
				} else {
					return
				}
			}

			// 取消息
			var msg []byte
			ok := make(chan struct{})
			select {
			case msg = <-c.WriteQueue:
				close(ok)
			case <-tools.WaitTimeout(ok, 30*time.Second):
				return
			case <-c.Done:
				return
			}

			// 写消息
			select {
			case <-c.Done:
				return
			default:
				if err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
					return
				}
				if _, err := c.Conn.Write(msg); err != nil {
					return
				}
				logger.Test("Write message: %v", msg)
			}
		}
	}
}
