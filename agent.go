package ecs_consul

import (
	"errors"
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"strings"
	"time"
)

const (
	keySeparator = "/"

	defaultHostPort          = 80
	defaultAgentHost         = "127.0.0.1"
	defaultAgentPort         = 8500
	defaultContainerPort     = 9000
	defaultGrpcContainerPort = 9090

	statusPass = "pass"
	statusFail = "fail"
)

var (
	errServiceName        = errors.New("service name is not provided")
	errServiceAddr        = errors.New("service address is not provided")
	errServiceEnv         = errors.New("service environment is not provided")
	errConfigNotValidated = errors.New("consul agent config has not been validated")
)

type Config struct {
	ServiceName   string        `json:"serviceName"`
	ContainerPort int           `json:"containerPort"`
	HostPort      int           `json:"hostPort"`
	TargetPort    int           `json:"targetPort"`
	Address       string        `json:"address"`
	TTL           time.Duration `json:"ttl"`
	Env           string        `json:"env"`
	ConsulAddress string        `json:"consulAddress"`
	AgentPort     int           `json:"agentPort"`
	BehindProxy   bool          `json:"behindProxy"`
	PassPhrase    string        `json:"passPhrase"`
	FailPhrase    string        `json:"failPhrase"`

	validated bool
}

type Agent struct {
	config  *Config
	agent   *consul.Agent
	catalog *consul.Catalog
	kv      *consul.KV
	client  *consul.Client
}

type CheckFunc func() bool

func (c *Config) MapPorts(mapping []PortMapping) {
	for i := range mapping {
		if mapping[i].ContainerPort == c.ContainerPort {
			c.HostPort = mapping[i].HostPort
			c.Address = mapping[i].BindIP
			break
		}
	}
}

func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return errServiceName
	}

	if c.Env == "" {
		return errServiceEnv
	}

	if c.Address == "" {
		return errServiceAddr
	}

	if c.ConsulAddress == "" {
		c.ConsulAddress = c.Address
	}

	if c.ContainerPort == 0 {
		c.ContainerPort = defaultContainerPort
	}

	if c.HostPort == 0 {
		c.HostPort = defaultHostPort
	}

	if c.TargetPort == 0 {
		if c.BehindProxy {
			c.TargetPort = c.HostPort
		} else {
			c.TargetPort = c.ContainerPort
		}
	}

	if c.TTL == 0 {
		c.TTL = time.Second * 15
	}

	if c.AgentPort == 0 {
		c.AgentPort = defaultAgentPort
	}

	if c.PassPhrase == "" {
		c.PassPhrase = "Service alive and reachable."
	}

	if c.FailPhrase == "" {
		c.FailPhrase = "Service unreachable."
	}

	c.validated = true

	return nil
}

func NewAgent(config *Config) (*Agent, error) {
	if !config.validated {
		return nil, errConfigNotValidated
	}

	s := &Agent{config: config}
	err := s.newClient()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (a *Agent) KV() *consul.KV {
	return a.kv
}

func (a *Agent) Client() *consul.Client {
	return a.client
}

func (a *Agent) Agent() *consul.Agent {
	return a.agent
}

func (a *Agent) Catalog() *consul.Catalog {
	return a.catalog
}

func (a *Agent) UpdateTTL(check CheckFunc) {
	ticker := time.NewTicker(a.config.TTL / 2)
	for range ticker.C {
		if err := a.update(check); err != nil {
			return
		}
	}
}

func (a *Agent) update(check CheckFunc) error {
	if !check() {
		if err := a.agent.UpdateTTL(a.formatCheckID(), a.config.FailPhrase, statusFail); err != nil {
			return err
		}
	}

	return a.agent.UpdateTTL(a.formatCheckID(), a.config.PassPhrase, statusPass)
}

func (a *Agent) newClient() error {
	client, err := consul.NewClient(&consul.Config{
		Address: fmt.Sprintf("%s:%d", a.config.ConsulAddress, a.config.AgentPort),
	})
	if err != nil {
		return err
	}
	a.client = client
	a.catalog = client.Catalog()
	a.agent = client.Agent()
	a.kv = client.KV()
	return nil
}

func (a *Agent) formatCheckID() string {
	return fmt.Sprintf("service:%s", a.config.ServiceName)
}

func (a *Agent) formatPrefix() string {
	return fmt.Sprintf("%s/%s/", a.config.ServiceName, a.config.Env)
}

func (a *Agent) formatCredential(cred string) string {
	return fmt.Sprintf("%s/%s/%s", a.config.ServiceName, a.config.Env, cred)
}

func (a *Agent) replaceKey(key string) string {
	parts := strings.Split(key, keySeparator)

	return parts[len(parts)-1]
}
