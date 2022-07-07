package main

import (
	"bufio"
	"fmt"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
	"github.com/inconshreveable/log15"
	"os"
	"strings"
	"sync"
	"time"
)

var log = log15.New()

var bridgeCallTypes = make(map[string]string)
var bridgeNumOfCalls = make(map[string]int)
var channels = make(map[string]*ari.ChannelHandle)
var bridges = make(map[string]*ari.BridgeHandle)
var endpoints = make(map[string]string)
var lock sync.Mutex

func main() {

	log.Info("Connecting")

	cl, err := native.Connect(&native.Options{
		Application:  "ARI",
		Username:     "asterisk",
		Password:     "asterisk123&",
		URL:          "http://localhost:8088/ari",
		WebsocketURL: "ws://localhost:8088/ari/events",
	})
	if err != nil {
		log.Error("Failed to build native ARI client", "error", err)
		return
	}
	defer cl.Close()
	log.Info("Connected to asterisk")

out:
	for {
		command, parameters := input()
		switch command {
		case "dial":
			if numOfParameters(parameters) {
				dial(cl, parameters)
			}
		case "list":
			fmt.Println("Listing all available calls:")
			list()
		case "join":
			if numOfParameters(parameters) {
				join(cl, parameters[0], parameters[1:])
			}
		case "exit":
			break out
		default:
			log.Error("Invalid command, please try again")
		}
	}
}

func join(cl ari.Client, bridgeID string, endpoints []string) {
	if bridge, available := bridges[bridgeID]; available {
		newEndpoints(cl, endpoints, bridge)
	} else {
		log.Error("CallID is not available")
		return
	}
}

func list() {
	if len(bridges) < 1 {
		log.Info("There are currently no active calls")
		return
	}

	for _, bridge := range bridges {
		bridgeData, err := bridge.Data()
		if err != nil {
			log.Error("Unable to access bridge")
			return
		}
		bridgeChannels := bridgeData.ChannelIDs
		fmt.Println("The call with ID: ", bridge.ID(), " Type: ", bridgeCallTypes[bridge.ID()], ", call has the following participants: ")

		for _, channel := range bridgeChannels {
			fmt.Println(endpoints[channel], "   ")
		}
		fmt.Println()
	}
}

func dial(cl ari.Client, parameters []string) {
	typeOfCall := typeOfCall(parameters)

	bridge := creatingBridge(cl, typeOfCall)

	newEndpoints(cl, parameters, bridge)

	go call(bridge)
}

func typeOfCall(parameters []string) string {
	if len(parameters) == 2 {
		return "call"
	} else {
		return "conference"
	}
}

func creatingBridge(cl ari.Client, typeOfCall string) *ari.BridgeHandle {
	bridge, err := cl.Bridge().Create(&ari.Key{}, "mixing", "")
	if err != nil {
		log.Info("Failed to create bridge for provided endpoints, error: ", err)
		return nil
	}
	bridges[bridge.ID()] = bridge
	bridgeCallTypes[bridge.ID()] = typeOfCall
	bridgeNumOfCalls[bridge.ID()] = 0
	fmt.Println("Bridge created: ", *bridge, ", type: ", typeOfCall)

	return bridge
}

func newEndpoints(cl ari.Client, parameters []string, bridge *ari.BridgeHandle) {
	for _, endpoint := range parameters {
		newChannel(cl, endpoint, bridge)
	}
}

func newChannel(cl ari.Client, endpoint string, bridge *ari.BridgeHandle) {
	channel, err := cl.Channel().Create(&ari.Key{}, ari.ChannelCreateRequest{
		Endpoint: "PJSIP/" + endpoint,
		App:      "ARI",
	})

	if err != nil {
		log.Error("Failed to create channel, error:", err)
		return
	} else {
		log.Info("Channel successfully created for: ", endpoint)
	}

	err = channel.Dial("ARI", 30*time.Second)
	if err != nil {
		log.Error("Dial has failed", err)
		return
	}

	err = bridge.AddChannel(channel.ID())
	if err != nil {
		log.Error("Adding channel to the bridge failed", err)
		return
	}

	bridgeNumOfCalls[bridge.ID()]++
	channels[channel.ID()] = channel
	endpoints[channel.ID()] = "PJSIP/ " + endpoint

	if bridgeNumOfCalls[bridge.ID()] > 2 {
		bridgeCallTypes[bridge.ID()] = "conference"
	}

	if _, err := bridge.Play(bridge.ID(), "sound:confbridge-join"); err != nil {
		log.Error("failed to play join sound", "error", err)
	}
}

func call(bridge *ari.BridgeHandle) {
	leaveSub := bridge.Subscribe(ari.Events.ChannelLeftBridge)

	for {
		events := <-leaveSub.Events()
		_, okay := events.(*ari.ChannelLeftBridge)

		if !okay {
			log.Error("Unable to destroy channels and bridge")
			return
		}

		isBreak := destroy(bridge)

		if isBreak {
			break
		}
	}
}

func destroy(bridge *ari.BridgeHandle) bool {
	lock.Lock()
	defer lock.Unlock()
	bridge.Play(bridge.ID(), "sound:confbridge-leave")
	bridgeNumOfCalls[bridge.ID()]--

	numberOfChannels := bridgeNumOfCalls[bridge.ID()]
	typeOfCall := bridgeCallTypes[bridge.ID()]

	if typeOfCall == "call" || numberOfChannels < 1 {
		deletingBridge(bridge)
		return true
	}
	return false
}

func deletingBridge(bridge *ari.BridgeHandle) {
	deletingChannels(bridge)
	bridge.Delete()
	delete(bridgeCallTypes, bridge.ID())
	delete(bridgeNumOfCalls, bridge.ID())
	delete(bridges, bridge.ID())
}

func deletingChannels(bridge *ari.BridgeHandle) {
	bridgeData, err := bridge.Data()
	if err != nil {
		log.Error("Unable to access bridge data")
		return
	}
	channelsIDs := bridgeData.ChannelIDs

	for _, channelID := range channelsIDs {
		err := channels[channelID].Hangup()
		delete(channels, channelID)
		if err != nil {
			log.Error("Unable to destroy remaining channel")
		}
	}
}

func input() (command string, parameters []string) {
	fmt.Print("Please enter your choice: ")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	parameters = strings.Fields(text)
	return parameters[0], parameters[1:]
}

func numOfParameters(parameters []string) bool {
	if len(parameters) < 2 {
		log.Error("they are not enough arguments")
		return false
	} else {
		return true
	}
}
