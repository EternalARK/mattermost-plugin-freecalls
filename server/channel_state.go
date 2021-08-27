package main

import (
	"encoding/json"
	"fmt"
)

type channelState struct {
	NodeID  string              `json:"node_id,omitempty"`
	Users   map[string]struct{} `json:"users,omitempty"`
	Enabled bool                `json:"enabled"`
}

func (p *Plugin) kvGetChannelState(channelID string) (*channelState, error) {
	data, appErr := p.API.KVGet(channelID)
	if appErr != nil {
		return nil, fmt.Errorf("KVGet failed: %w", appErr)
	}
	if data == nil {
		return nil, nil
	}
	var state *channelState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return state, nil
}

func (p *Plugin) kvSetAtomicChannelState(channelID string, cb func(state *channelState) (*channelState, error)) error {
	return p.kvSetAtomic(channelID, func(data []byte) ([]byte, error) {
		var err error
		var state *channelState
		if data != nil {
			if err := json.Unmarshal(data, &state); err != nil {
				return nil, err
			}
		}
		state, err = cb(state)
		if err != nil {
			return nil, err
		}
		if state == nil {
			return nil, nil
		}
		return json.Marshal(state)
	})
}

func (p *Plugin) cleanUpState() error {
	var page int
	perPage := 100
	for {
		keys, appErr := p.API.KVList(page, perPage)
		if appErr != nil {
			return appErr
		}
		if len(keys) == 0 {
			break
		}
		for _, k := range keys {
			if err := p.kvSetAtomicChannelState(k, func(state *channelState) (*channelState, error) {
				if state == nil {
					return nil, nil
				}
				state.NodeID = ""
				state.Users = nil
				return state, nil
			}); err != nil {
				return fmt.Errorf("failed to clean up state: %w", err)
			}
		}
		page++
	}
	return nil
}
