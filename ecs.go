package ecs_consul

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type PortMapping struct {
	ContainerPort int    `json:"ContainerPort"`
	HostPort      int    `json:"HostPort"`
	BindIP        string `json:"BindIp"`
	Protocol      string `json:"Protocol"`
}

type ecsTaskMetadata struct {
	PortMappings []PortMapping
}

func GetPortMappings() ([]PortMapping, error) {
	return readInstanceMetadataFile(os.Getenv("ECS_CONTAINER_METADATA_FILE"))
}

func readInstanceMetadataFile(path string) ([]PortMapping, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var metadata ecsTaskMetadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON in %s: %w", path, err)
	}

	return metadata.PortMappings, nil
}
