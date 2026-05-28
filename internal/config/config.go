package config

import (
	log "github.com/sirupsen/logrus"

	"github.com/kelseyhightower/envconfig"
)

type Camara struct {
	HostAgentAddr string `split_words:"true" default:"0.0.0.0:8080"`
	LogLevel      string `split_words:"true" default:"info"`
	ApiRoot       string `split_words:"true" default:"nearbyone.operator-name.nearbycomputing.com"`
}

type Controller struct {
	Namespace string `split_words:"true" required:"true"`
}

type Config struct {
	Camara
	Controller
}

func process(prefix string, spec interface{}) {
	if err := envconfig.Process(prefix, spec); err != nil {
		log.Fatal(err.Error())
	}
}

func GetConf() Config {
	var camara Camara
	process("camara", &camara)

	var controller Controller
	process("controller", &controller)

	return Config{camara, controller}
}
