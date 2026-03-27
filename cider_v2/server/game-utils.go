package server

func (gm *GameManager) handlePublicCmd(cmd *Command) {
	switch cmd.CmdType {
	case Handover:
		session, found := gm.Sessions[cmd.Id]
		if !found {
			warnLogger.Printf("got a handover cmd from a session routine but could not find it: %s\n", cmd.Id)
			return
		}

		lastPlayer := session.State.lastPlayerId
		for _, p := range session.Players {
			if p.String() != lastPlayer {
				session.State.lastPlayerId = p.String()
				infoLogger.Println("updated last player id after handover cmd")
				return
			}
		}

	}
}
