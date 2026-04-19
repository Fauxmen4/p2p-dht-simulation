package simulation

import (
	"fmt"
	"my-kad-dht/core/network"
	"path"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

func ConfigBased(configName string) {
	config := MustLoad(configName)
	fmt.Println(config)

    net := network.New(config.Kademlia)

}
