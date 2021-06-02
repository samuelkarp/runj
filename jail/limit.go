package jail

import (
	"bytes"
	"os/exec"

	"go.sbk.wtf/runj/runtimespec"
)

// Limit uses rctl to add the rct rules
func Limit(id string, ociConfig *runtimespec.Spec) error {
	if ociConfig.FreeBSD == nil {
		return nil
	}
	for _, racctLimit := range ociConfig.FreeBSD.RacctLimits {
		rule := makeRCTLRule(id, &racctLimit)
		cmd := exec.Command("rctl", "-a", rule)
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

// Unlimit uses rctl to remove the rctl rules
func Unlimit(id string, ociConfig *runtimespec.Spec) error {
	if ociConfig.FreeBSD == nil {
		return nil
	}
	for _, racctLimit := range ociConfig.FreeBSD.RacctLimits {
		rule := makeRCTLRule(id, &racctLimit)
		cmd := exec.Command("rctl", "-r", rule)
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func makeRCTLRule(id string, racctLimit *runtimespec.RacctLimit) string {
	buf := bytes.Buffer{}
	buf.WriteString("jail:")
	buf.WriteString(id)
	buf.WriteString(":")
	buf.WriteString(racctLimit.Resource)
	buf.WriteString(":")
	buf.WriteString(racctLimit.Action)
	buf.WriteString("=")
	buf.WriteString(racctLimit.Amount)
	if racctLimit.Per != "" {
		buf.WriteString("/")
		buf.WriteString(racctLimit.Per)
	}
	return buf.String()
}
