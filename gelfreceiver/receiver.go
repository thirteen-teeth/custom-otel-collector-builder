package gelfreceiver

import (
	"context"
	"fmt"
	"net"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

// gelfReceiver implements the receiver.Logs interface
type gelfReceiver struct {
	config   *Config
	settings receiver.Settings
	consumer consumer.Logs

	// Server components
	tcpListener net.Listener
	udpConn     *net.UDPConn

	// Control
	cancel context.CancelFunc
	wg     sync.WaitGroup

	logger *zap.Logger
}

// Start implements component.Component
func (r *gelfReceiver) Start(ctx context.Context, host component.Host) error {
	r.logger = r.settings.Logger

	ctx, r.cancel = context.WithCancel(ctx)

	r.logger.Info("Starting GELF receiver",
		zap.String("listen_address", r.config.ListenAddress),
		zap.String("protocol", r.config.Protocol),
		zap.Bool("compression", r.config.UseCompression),
		zap.Int("max_message_size", r.config.MaxMessageSize))

	// Start TCP listener if configured
	if r.config.Protocol == "tcp" || r.config.Protocol == "both" {
		if err := r.startTCPListener(ctx); err != nil {
			r.logger.Error("Failed to start TCP listener", zap.Error(err))
			return fmt.Errorf("failed to start TCP listener: %w", err)
		}
	}

	// Start UDP listener if configured
	if r.config.Protocol == "udp" || r.config.Protocol == "both" {
		if err := r.startUDPListener(ctx); err != nil {
			r.logger.Error("Failed to start UDP listener", zap.Error(err))
			return fmt.Errorf("failed to start UDP listener: %w", err)
		}
	}

	r.logger.Info("GELF receiver started successfully")
	return nil
}

// Shutdown implements component.Component
func (r *gelfReceiver) Shutdown(ctx context.Context) error {
	r.logger.Info("Shutting down GELF receiver")

	if r.cancel != nil {
		r.cancel()
	}

	// Close TCP listener
	if r.tcpListener != nil {
		if err := r.tcpListener.Close(); err != nil {
			r.logger.Warn("Error closing TCP listener", zap.Error(err))
		}
	}

	// Close UDP connection
	if r.udpConn != nil {
		if err := r.udpConn.Close(); err != nil {
			r.logger.Warn("Error closing UDP connection", zap.Error(err))
		}
	}

	// Wait for goroutines to finish
	r.wg.Wait()

	r.logger.Info("GELF receiver shut down successfully")
	return nil
}

// startTCPListener starts the TCP listener
func (r *gelfReceiver) startTCPListener(ctx context.Context) error {
	listener, err := net.Listen("tcp", r.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on TCP %s: %w", r.config.ListenAddress, err)
	}

	r.tcpListener = listener
	r.logger.Info("TCP listener started", zap.String("address", listener.Addr().String()))

	r.wg.Add(1)
	go r.runTCPServer(ctx)

	return nil
}

// startUDPListener starts the UDP listener
func (r *gelfReceiver) startUDPListener(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", r.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address %s: %w", r.config.ListenAddress, err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP %s: %w", r.config.ListenAddress, err)
	}

	r.udpConn = conn
	r.logger.Info("UDP listener started", zap.String("address", conn.LocalAddr().String()))

	r.wg.Add(1)
	go r.runUDPServer(ctx)

	return nil
}

// runTCPServer handles TCP connections
func (r *gelfReceiver) runTCPServer(ctx context.Context) {
	defer r.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := r.tcpListener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					// Context cancelled, shutting down
					return
				}
				r.logger.Error("Failed to accept TCP connection", zap.Error(err))
				continue
			}

			// Handle connection in a separate goroutine
			r.wg.Add(1)
			go r.handleTCPConnection(ctx, conn)
		}
	}
}

// runUDPServer handles UDP packets
func (r *gelfReceiver) runUDPServer(ctx context.Context) {
	defer r.wg.Done()

	buffer := make([]byte, r.config.MaxMessageSize)
	chunkAssembler := NewChunkAssembler(r.config.ChunkTimeout, r.logger)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, addr, err := r.udpConn.ReadFromUDP(buffer)
			if err != nil {
				if ctx.Err() != nil {
					// Context cancelled, shutting down
					return
				}
				r.logger.Error("Failed to read UDP packet", zap.Error(err))
				continue
			}

			r.wg.Add(1)
			go r.handleUDPPacket(ctx, buffer[:n], addr, chunkAssembler)
		}
	}
}

// handleTCPConnection processes a single TCP connection
func (r *gelfReceiver) handleTCPConnection(ctx context.Context, conn net.Conn) {
	defer r.wg.Done()
	defer conn.Close()

	processor := NewTCPProcessor(r.config, r.consumer, r.logger)
	if err := processor.Process(ctx, conn); err != nil {
		r.logger.Error("Failed to process TCP connection",
			zap.Error(err),
			zap.String("remote_addr", conn.RemoteAddr().String()))
	}
}

// handleUDPPacket processes a single UDP packet
func (r *gelfReceiver) handleUDPPacket(ctx context.Context, data []byte, addr *net.UDPAddr, assembler *ChunkAssembler) {
	defer r.wg.Done()

	processor := NewUDPProcessor(r.config, r.consumer, r.logger)
	if err := processor.Process(ctx, data, addr, assembler); err != nil {
		r.logger.Error("Failed to process UDP packet",
			zap.Error(err),
			zap.String("remote_addr", addr.String()))
	}
}
