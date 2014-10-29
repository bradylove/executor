package exchanger

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/executor"
	garden "github.com/cloudfoundry-incubator/garden/api"
)

type GardenClient interface {
	Create(garden.ContainerSpec) (garden.Container, error)
	Destroy(handle string) error
	Containers(garden.Properties) ([]garden.Container, error)
	Lookup(handle string) (garden.Container, error)
}

type Exchanger interface {
	Garden2Executor(GardenClient, garden.Container) (executor.Container, error)
	Executor2Garden(GardenClient, executor.Container) (garden.Container, error)
}

const (
	tagPropertyPrefix      = "tag:"
	executorPropertyPrefix = "executor:"

	ContainerOwnerProperty       = executorPropertyPrefix + "owner"
	ContainerGuidProperty        = executorPropertyPrefix + "guid"
	ContainerStateProperty       = executorPropertyPrefix + "state"
	ContainerAllocatedAtProperty = executorPropertyPrefix + "allocated-at"
	ContainerRootfsProperty      = executorPropertyPrefix + "rootfs"
	ContainerCompleteURLProperty = executorPropertyPrefix + "complete-url"
	ContainerActionsProperty     = executorPropertyPrefix + "actions"
	ContainerEnvProperty         = executorPropertyPrefix + "env"
	ContainerLogProperty         = executorPropertyPrefix + "log"
	ContainerResultProperty      = executorPropertyPrefix + "result"
)

type InvalidStateError struct {
	State string
}

func (err InvalidStateError) Error() string {
	return fmt.Sprintf("invalid state: %s", err.State)
}

type InvalidTimestampError struct {
	Timestamp string
}

func (err InvalidTimestampError) Error() string {
	return fmt.Sprintf("invalid timestamp: %s", err.Timestamp)
}

type InvalidJSONError struct {
	Property     string
	Value        string
	UnmarshalErr error
}

func (err InvalidJSONError) Error() string {
	return fmt.Sprintf(
		"invalid JSON in property %s: %s\n\nvalue: %s",
		err.Property,
		err.UnmarshalErr.Error(),
		err.Value,
	)
}

type ContainerLookupError struct {
	Handle    string
	LookupErr error
}

func (err ContainerLookupError) Error() string {
	return fmt.Sprintf(
		"lookup container with handle %s failed: %s",
		err.Handle,
		err.LookupErr.Error(),
	)
}

func NewExchanger(
	containerOwnerName string,
	containerMaxCPUShares uint64,
	containerInodeLimit uint64,
) Exchanger {
	return exchanger{
		containerOwnerName:    containerOwnerName,
		containerMaxCPUShares: containerMaxCPUShares,
		containerInodeLimit:   containerInodeLimit,
	}
}

type exchanger struct {
	containerOwnerName    string
	containerMaxCPUShares uint64
	containerInodeLimit   uint64
}

func (exchanger exchanger) Garden2Executor(gardenClient GardenClient, gardenContainer garden.Container) (executor.Container, error) {
	info, err := gardenContainer.Info()
	if err != nil {
		return executor.Container{}, err
	}

	memoryLimits, err := gardenContainer.CurrentMemoryLimits()
	if err != nil {
		return executor.Container{}, err
	}

	diskLimits, err := gardenContainer.CurrentDiskLimits()
	if err != nil {
		return executor.Container{}, err
	}

	cpuLimits, err := gardenContainer.CurrentCPULimits()
	if err != nil {
		return executor.Container{}, err
	}

	executorContainer := executor.Container{
		MemoryMB:  int(memoryLimits.LimitInBytes / 1024 / 1024),

		DiskMB:    int(diskLimits.ByteHard / 1024 / 1024),
		CPUWeight: uint(100.0 * float64(cpuLimits.LimitInShares) / float64(exchanger.containerMaxCPUShares)),

		Tags:  executor.Tags{},
		Ports: make([]executor.PortMapping, len(info.MappedPorts)),

		ContainerHandle: gardenContainer.Handle(),
	}

	for key, value := range info.Properties {
		switch key {
		case ContainerGuidProperty:
			executorContainer.Guid = value
		case ContainerStateProperty:
			state := executor.State(value)

			if state == executor.StateReserved ||
				state == executor.StateInitializing ||
				state == executor.StateCreated ||
				state == executor.StateCompleted {
				executorContainer.State = state
			} else {
				return executor.Container{}, InvalidStateError{value}
			}
		case ContainerAllocatedAtProperty:
			_, err := fmt.Sscanf(value, "%d", &executorContainer.AllocatedAt)
			if err != nil {
				return executor.Container{}, InvalidTimestampError{value}
			}
		case ContainerRootfsProperty:
			executorContainer.RootFSPath = value
		case ContainerCompleteURLProperty:
			executorContainer.CompleteURL = value
		case ContainerActionsProperty:
			err := json.Unmarshal([]byte(value), &executorContainer.Actions)
			if err != nil {
				return executor.Container{}, InvalidJSONError{
					Property:     key,
					Value:        value,
					UnmarshalErr: err,
				}
			}
		case ContainerEnvProperty:
			err := json.Unmarshal([]byte(value), &executorContainer.Env)
			if err != nil {
				return executor.Container{}, InvalidJSONError{
					Property:     key,
					Value:        value,
					UnmarshalErr: err,
				}
			}
		case ContainerLogProperty:
			err := json.Unmarshal([]byte(value), &executorContainer.Log)
			if err != nil {
				return executor.Container{}, InvalidJSONError{
					Property:     key,
					Value:        value,
					UnmarshalErr: err,
				}
			}
		case ContainerResultProperty:
			err := json.Unmarshal([]byte(value), &executorContainer.RunResult)
			if err != nil {
				return executor.Container{}, InvalidJSONError{
					Property:     key,
					Value:        value,
					UnmarshalErr: err,
				}
			}
		default:
			if strings.HasPrefix(key, tagPropertyPrefix) {
				executorContainer.Tags[key[len(tagPropertyPrefix):]] = value
			}
		}
	}

	for i, mapping := range info.MappedPorts {
		executorContainer.Ports[i] = executor.PortMapping{
			HostPort:      mapping.HostPort,
			ContainerPort: mapping.ContainerPort,
		}
	}

	return executorContainer, nil
}

func (exchanger exchanger) Executor2Garden(gardenClient GardenClient, executorContainer executor.Container) (garden.Container, error) {
	containerSpec := garden.ContainerSpec{
		Handle:     executorContainer.ContainerHandle,
		RootFSPath: executorContainer.RootFSPath,
	}

	actionsJson, err := json.Marshal(executorContainer.Actions)
	if err != nil {
		return nil, err
	}

	envJson, err := json.Marshal(executorContainer.Env)
	if err != nil {
		return nil, err
	}

	logJson, err := json.Marshal(executorContainer.Log)
	if err != nil {
		return nil, err
	}

	resultJson, err := json.Marshal(executorContainer.RunResult)
	if err != nil {
		return nil, err
	}

	containerSpec.Properties = garden.Properties{
		ContainerOwnerProperty:       exchanger.containerOwnerName,
		ContainerGuidProperty:        executorContainer.Guid,
		ContainerStateProperty:       string(executorContainer.State),
		ContainerAllocatedAtProperty: fmt.Sprintf("%d", executorContainer.AllocatedAt),
		ContainerRootfsProperty:      executorContainer.RootFSPath,
		ContainerCompleteURLProperty: executorContainer.CompleteURL,
		ContainerActionsProperty:     string(actionsJson),
		ContainerEnvProperty:         string(envJson),
		ContainerLogProperty:         string(logJson),
		ContainerResultProperty:      string(resultJson),
	}

	for name, value := range executorContainer.Tags {
		containerSpec.Properties[tagPropertyPrefix+name] = value
	}

	gardenContainer, err := gardenClient.Create(containerSpec)
	if err != nil {
		return nil, err
	}

	for _, ports := range executorContainer.Ports {
		_, _, err := gardenContainer.NetIn(ports.HostPort, ports.ContainerPort)
		if err != nil {
			return nil, err
		}
	}

	if executorContainer.MemoryMB != 0 {
		err := gardenContainer.LimitMemory(garden.MemoryLimits{
			LimitInBytes: uint64(executorContainer.MemoryMB * 1024 * 1024),
		})
		if err != nil {
			return nil, err
		}
	}

	err = gardenContainer.LimitDisk(garden.DiskLimits{
		ByteHard:  uint64(executorContainer.DiskMB * 1024 * 1024),
		InodeHard: exchanger.containerInodeLimit,
	})
	if err != nil {
		return nil, err
	}

	err = gardenContainer.LimitCPU(garden.CPULimits{
		LimitInShares: uint64(float64(exchanger.containerMaxCPUShares) * float64(executorContainer.CPUWeight) / 100.0),
	})
	if err != nil {
		return nil, err
	}

	return gardenContainer, nil
}
