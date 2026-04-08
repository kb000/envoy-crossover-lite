package controller

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/mumoshu/crossover/pkg/kubeclient"
	"github.com/mumoshu/crossover/pkg/reconciler"
)

type Manager struct {
	Namespace    string
	Noop         bool
	Token        string
	Insecure     bool
	Server       string
	Watch        bool
	SyncInterval time.Duration
	OutputDir    string
	Onetime      bool
	ConfigMaps   StringSlice
}

func (m *Manager) Run(ctx context.Context) error {
	controllers := []*Controller{}

	cmclient := &kubeclient.KubeClient{
		Resource:     "configmaps",
		GroupVersion: "api/v1",
		Server:       m.Server,
		Token:        m.Token,
		HttpClient:   createHttpClient(m.Insecure),
	}

	configmaps := &Controller{
		updated:   make(chan string),
		namespace: m.Namespace,
		client:    cmclient,
		reconciler: &reconciler.ConfigmapReconciler{
			Client:    cmclient,
			Namespace: m.Namespace,
			OutputDir: m.OutputDir,
		},
		resourceNames: m.ConfigMaps,
	}

	controllers = append(controllers, configmaps)

	if m.Onetime {
		for i := range controllers {
			c := controllers[i]
			if err := c.Once(); err != nil {
				return err
			}
		}
		return nil
	}

	log.Println("Starting crossover...")

	var wg sync.WaitGroup

	for i := range controllers {
		c := controllers[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.Poll(ctx, m.SyncInterval); err != nil {
				log.Fatalf("%v", err)
			}
		}()
	}

	if m.Watch {
		for i := range controllers {
			c := controllers[i]
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := c.Watch(ctx); err != nil {
					log.Fatalf("Watch stopped due to error: %v", err)
				}
				log.Printf("Watch stopped normally.")
			}()
		}
	}

	for i := range controllers {
		c := controllers[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.Run(ctx); err != nil {
				log.Fatalf("Run loop stopped due to error: %v", err)
			}
			log.Printf("Run loop stopped normally.")
		}()
	}

	wg.Wait()

	return nil
}

func createHttpClient(insecure bool) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}
	client := &http.Client{
		Transport: transport,
	}
	return client
}
