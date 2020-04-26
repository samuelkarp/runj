package jail

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

const (
	defaultStateDir = "/var/lib/runj/jails"
	stateDir        = defaultStateDir
	confName        = "jail.conf"
	configTemplate  = `{{ .Name }} {
  path = "{{ .Root }}";
  persist;
}
`
)

func CreateConfig(id, root string) error {
	config, err := renderConfig(id, root)
	if err != nil {
		return err
	}
	fmt.Println(config)
	jailPath := filepath.Join(stateDir, id)
	err = os.MkdirAll(jailPath, 0755)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(jailPath, confName), []byte(config), 0644)
}

func renderConfig(id, root string) (string, error) {
	config, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return "", err
	}
	buf := bytes.Buffer{}
	config.Execute(&buf, struct {
		Name string
		Root string
	}{
		Name: id,
		Root: root,
	})
	return buf.String(), nil
}
