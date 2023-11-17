package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config for servers.
type Config struct {
	Servers []*Server `yaml:"servers"`
}

// Server is the ssh sever config.
type Server struct {
	Alias          []string        `yaml:"alias"`
	Host           string          `yaml:"host"`
	Port           uint            `yaml:"port"`
	User           string          `yaml:"user"`
	Password       string          `yaml:"password"`
	IdleMaxSeconds int             `yaml:"idle_max_seconds"`
	IdleSendString string          `yaml:"idle_send_string"`
	Expect         []*expectConfig `yaml:"expect"`

	passwordSent bool
	expectEnd    bool
}

type expectConfig struct {
	Match        string `yaml:"match"`
	Send         string `yaml:"send"`
	SendMaxTimes int    `yaml:"send_max_times"` // default 1
	End          bool   `yaml:"end"`

	match     *regexp.Regexp
	sendTimes int
}

func loadConfig(path string) (*Config, error) {
	c := &Config{}

	if path == "" {
		return c, nil
	}

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return c, nil
	}

	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}

	for _, v := range c.Servers {
		if v.Port == 0 {
			v.Port = 22
		}

		for _, ex := range v.Expect {
			if ex.Match == "" {
				return nil, fmt.Errorf("server: %s@%s, `expect.match` must not be empty", v.User, v.Host)
			}
			ex.match, err = regexp.Compile(ex.Match)
			if err != nil {
				return nil, fmt.Errorf("server: %s@%s, compile regexp `%s` of `expect.match` err: %v", v.User, v.Host, ex.Match, err)
			}

			if ex.SendMaxTimes == 0 {
				ex.SendMaxTimes = 1
			}
		}
	}

	return c, nil
}

func (c *Config) findServer(dest string) (*Server, error) {
	userHost := strings.Split(dest, "@")
	if len(userHost) == 2 {
		user := userHost[0]
		host := userHost[1]
		for _, v := range c.Servers {
			if v.Host == host && v.User == user {
				return v, nil
			}
		}
		return &Server{User: user, Host: host}, nil
	}

	var a []string
	for _, v := range c.Servers {
		for _, alias := range v.Alias {
			if alias == dest {
				return v, nil
			}
			a = append(a, alias)
		}
	}

	return nil, fmt.Errorf("not found alias `%s`, avaliable: [%s]", dest, strings.Join(a, " "))
}

func (c *Config) printAliases() {
	var a []string
	for _, v := range c.Servers {
		s := fmt.Sprintf("%s --> %s@%s", strings.Join(v.Alias, " "), v.User, v.Host)
		if v.Port > 0 && v.Port != 22 {
			s = fmt.Sprintf("%s [%d]", s, v.Port)
		}
		a = append(a, s)
	}
	if len(a) == 0 {
		fmt.Println("no alias configured")
	} else {
		fmt.Println(strings.Join(a, "\n"))
	}
}
