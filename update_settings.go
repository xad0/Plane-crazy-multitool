package main

func defaultUpdateChannel() string {
	if Version == "dev" {
		return updateChannelWorkshop
	}

	return updateChannelEdge
}

func updateChannelName(channel string) string {
	switch channel {
	case updateChannelWorkshop:
		return "Workshop"
	case updateChannelEdge:
		return "Edge"
	case updateChannelIronclad:
		return "Ironclad"
	default:
		return "Unknown"
	}
}

func updateChannelDescription(channel string) string {
	switch channel {
	case updateChannelWorkshop:
		return "Developer/source build. Manual update checks only."
	case updateChannelEdge:
		return "Current public release. Gets every release."
	case updateChannelIronclad:
		return "Conservative track. Gets important and critical releases."
	default:
		return "Unknown update channel."
	}
}

func updateChannelKeys() []string {
	return []string{
		updateChannelWorkshop,
		updateChannelEdge,
		updateChannelIronclad,
	}
}

func (m *model) setUpdateChannel(channel string) {
	if channel != updateChannelWorkshop &&
		channel != updateChannelEdge &&
		channel != updateChannelIronclad {
		channel = defaultUpdateChannel()
	}

	m.cfg.UpdateChannel = channel

	m.saveSettings()
}

func (m *model) cycleUpdateChannel(delta int) {
	keys := updateChannelKeys()

	current := 0

	for i, key := range keys {
		if m.cfg.UpdateChannel == key {
			current = i
			break
		}
	}

	current += delta

	for current < 0 {
		current += len(keys)
	}

	current %= len(keys)

	m.setUpdateChannel(keys[current])
}
