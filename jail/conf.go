package jail

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"go.sbk.wtf/runj/state"
)

const (
	confName       = "jail.conf"
	configTemplate = `{{ .Name }} {
  path = "{{ .Root }}";
{{- if ne .Hostname "" }}
  host.hostname = "{{.Hostname}}";
{{- end }}
{{- if ne .IP4 "" }}
  ip4 = "{{.IP4}}";
{{- end }}
{{- if gt (len .IP4Addr) 0 }}
  ip4.addr = {{ join .IP4Addr ", " }};
{{- end }}
{{- if ne .VNet "" }}
  vnet = "{{.VNet}}";
{{- end }}
{{- if gt (len .VNetInterface) 0 }}
  vnet.interface = "{{ join .VNetInterface ", " }}";
{{- end }}
  persist;
}
`
)

// Config is a limited subset of the parameters available in jail.conf(5) for use with jail(8).
type Config struct {
	Name          string
	Root          string
	Hostname      string
	IP4           string
	IP4Addr       []string
	VNet          string
	VNetInterface []string
}

// CreateConfig creates a config file for the jail(8) command
func CreateConfig(config *Config) (string, error) {
	cfg, err := renderConfig(config)
	if err != nil {
		return "", err
	}
	confPath := ConfPath(config.Name)
	confFile, err := os.OpenFile(confPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("jail: config should not already exist: %w", err)
	}
	defer func() {
		confFile.Close()
		if err != nil {
			os.Remove(confFile.Name())
		}
	}()
	_, err = confFile.Write([]byte(cfg))
	if err != nil {
		return "", err
	}
	return confFile.Name(), nil
}

// ConfPath returns the expected file path for a given jail's config file
func ConfPath(id string) string {
	return filepath.Join(state.Dir(id), confName)
}

func renderConfig(config *Config) (string, error) {
	cfg, err := template.New("config").Funcs(map[string]interface{}{
		"join": func(elems []string, sep string) string {
			return strings.Join(elems, sep)
		},
	}).Parse(configTemplate)
	if err != nil {
		return "", err
	}
	buf := bytes.Buffer{}
	cfg.Execute(&buf, config)
	return buf.String(), nil
}
