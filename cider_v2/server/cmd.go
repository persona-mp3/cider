package server

import (
	pb "github.com/persona-mp3/protocols/gen"
)

type InternalCommand int

const (
	Deliver InternalCommand = iota
	// Tell the game server to switch turns among particular set of players
	Handover
)

var gameServerId = "gameServeId01"

type Command struct {
	Id      string
	CmdType InternalCommand
	Packet  *pb.Packet
	data    any
}

func deliverCommand(packet *pb.Packet) *Command {
	return &Command{
		Id:      "gameServeId01",
		CmdType: Deliver,
		Packet:  packet,
	}
}
