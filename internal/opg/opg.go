package opg

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"sync"

	opgc "github.com/nbycomp/neonephos-opg-ewbi-api/api/federation/client"
)

type OPGClientsMap struct {
	opgClients         map[string]opgc.ClientWithResponsesInterface
	mutex              *sync.Mutex
	insecureSkipVerify bool
}

type OPGClientsMapInterface interface {
	GetOPGClient(fedId, url, client string) opgc.ClientWithResponsesInterface
	SetOPGClient(key string, m opgc.ClientWithResponsesInterface)
}

type OPGClientsMapOpt func(*OPGClientsMap)

func NewOPGClientsMap(opts ...OPGClientsMapOpt) OPGClientsMapInterface {
	m := &OPGClientsMap{
		opgClients: map[string]opgc.ClientWithResponsesInterface{},
		mutex:      &sync.Mutex{},
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

func WithInsecureSkipVerify() OPGClientsMapOpt {
	return func(m *OPGClientsMap) {
		m.insecureSkipVerify = true
	}
}

func WithInsecureSkipVerifyClientOption(insecureSkipVerify bool) opgc.ClientOption {
	return func(c *opgc.Client) error {

		if c.Client != nil {
			return errors.New("opgClient already exists and may not support the insecureSkipVerify interface")
		}

		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecureSkipVerify}
		c.Client = &http.Client{Transport: tr}

		return nil
	}
}

func (m *OPGClientsMap) GetOPGClient(fedId, url, client string) opgc.ClientWithResponsesInterface {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, ok := m.opgClients[fedId]
	if !ok {

		opts := []opgc.ClientOption{
			opgc.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
				req.Header.Add("X-Client-ID", client)
				return nil
			}),
		}
		if m.insecureSkipVerify {
			opts = append(opts, WithInsecureSkipVerifyClientOption(m.insecureSkipVerify))
		}

		newC, _ := opgc.NewClientWithResponses(
			url,
			opts...,
		)
		m.opgClients[fedId] = newC
	}
	return m.opgClients[fedId]
}

func (m *OPGClientsMap) SetOPGClient(key string, c opgc.ClientWithResponsesInterface) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.opgClients[key] = c
}
