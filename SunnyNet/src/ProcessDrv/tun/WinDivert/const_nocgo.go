//go:build windows && !cgo
// +build windows,!cgo

package WinDivert

const (
	LayerNetwork        = Layer(0)
	LayerNetworkForward = Layer(1)
	LayerFlow           = Layer(2)
	LayerSocket         = Layer(3)
	LayerReflect        = Layer(4)
)

const (
	EventNetworkPacket   = Event(0)
	EventFlowEstablished = Event(1)
	EventFlowDeleted     = Event(2)
	EventSocketBind      = Event(3)
	EventSocketConnect   = Event(4)
	EventSocketListen    = Event(5)
	EventSocketAccept    = Event(6)
	EventSocketClose     = Event(7)
	EventReflectOpen     = Event(8)
	EventReflectClose    = Event(9)
)

const (
	ShutdownRecv = Shutdown(1)
	ShutdownSend = Shutdown(2)
	ShutdownBoth = Shutdown(3)
)

const (
	QueueLength  = Param(0)
	QueueTime    = Param(1)
	QueueSize    = Param(2)
	VersionMajor = Param(3)
	VersionMinor = Param(4)
)

const (
	FlagDefault   = uint64(0)
	FlagSniff     = uint64(1)
	FlagDrop      = uint64(2)
	FlagRecvOnly  = uint64(4)
	FlagSendOnly  = uint64(8)
	FlagNoInstall = uint64(16)
	FlagFragments = uint64(32)
)

const (
	PriorityDefault    = int16(0)
	PriorityHighest    = int16(30000)
	PriorityLowest     = int16(-30000)
	QueueLengthDefault = uint64(4096)
	QueueLengthMin     = uint64(32)
	QueueLengthMax     = uint64(16384)
	QueueTimeDefault   = uint64(2000)
	QueueTimeMin       = uint64(100)
	QueueTimeMax       = uint64(16000)
	QueueSizeDefault   = uint64(4194304)
	QueueSizeMin       = uint64(65535)
	QueueSizeMax       = uint64(33554432)
)

const (
	ChecksumDefault  = uint64(0)
	NoIPChecksum     = uint64(1)
	NoICMPChecksum   = uint64(2)
	NoICMPV6Checksum = uint64(4)
	NoTCPChecksum    = uint64(8)
	NoUDPChecksum    = uint64(16)
)

const (
	BatchMax = int(64)
	MTUMax   = int(40 + 0xFFFF)
)
