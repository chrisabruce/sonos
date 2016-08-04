package main

import (
	//"fmt"
	"github.com/chrisabruce/sonos"
)

const SPEAKER_IP = "10.0.1.32"

func main() {

	zp := sonos.NewZonePlayer(SPEAKER_IP)
	zp.Stop()

}
