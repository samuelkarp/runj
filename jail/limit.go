package jail

import (
	"fmt"
	"os/exec"

	"go.sbk.wtf/runj/runtimespec"
)

// Limit uses rctl to add the rct rules
func Limit(id string, ociConfig *runtimespec.Spec) error {
	if ociConfig.FreeBSD == nil {
		return nil
	}
	for _, rule := range makeRCTLRules(id, ociConfig.FreeBSD.Resources) {
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
	for _, rule := range makeRCTLRules(id, ociConfig.FreeBSD.Resources) {
		cmd := exec.Command("rctl", "-r", rule)
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func makeRCTLMemoryRules(id string, memory *runtimespec.FreeBSDMemory) []string {
	var rules []string
	if memory.Limit != nil {
		rules = append(rules, formatRCTLRule(id, "memoryuse", "deny", *memory.Limit))
	}
	if memory.Warning != nil {
		rules = append(rules, formatRCTLRule(id, "memoryuse", "devctl", *memory.Warning))
	}
	if memory.Swap != nil {
		rules = append(rules, formatRCTLRule(id, "swapuse", "deny", *memory.Swap))
	}
	if memory.SwapWarning != nil {
		rules = append(rules, formatRCTLRule(id, "swapuse", "devctl", *memory.SwapWarning))
	}
	return rules
}

func makeRCTLFSIORules(id string, fsio *runtimespec.FreeBSDFSIO) []string {
	var rules []string
	if fsio.ReadBPS != nil {
		rules = append(rules, formatRCTLRule(id, "readbps", "throttle", *fsio.ReadBPS))
	}
	if fsio.WriteBPS != nil {
		rules = append(rules, formatRCTLRule(id, "writebps", "throttle", *fsio.WriteBPS))
	}
	if fsio.ReadIOPS != nil {
		rules = append(rules, formatRCTLRule(id, "readiops", "throttle", *fsio.ReadIOPS))
	}
	if fsio.WriteIOPS != nil {
		rules = append(rules, formatRCTLRule(id, "writeiops", "throttle", *fsio.WriteIOPS))
	}
	return rules
}

func makeRCTLShmRules(id string, shm *runtimespec.FreeBSDShm) []string {
	var rules []string
	if shm.Count != nil {
		rules = append(rules, formatRCTLRule(id, "nshm", "deny", *shm.Count))
	}
	if shm.Size != nil {
		rules = append(rules, formatRCTLRule(id, "shmsize", "deny", *shm.Size))
	}
	return rules
}

func makeRCTLCPURules(id string, cpu *runtimespec.FreeBSDCPU) []string {
	var rules []string
	if cpu.Limit != nil {
		rules = append(rules, formatRCTLRule(id, "pcpu", "deny", *cpu.Limit))
	}
	return rules
}

func makeRCTLProcessRules(id string, proc *runtimespec.FreeBSDProcess) []string {
	var rules []string
	if proc.Limit != nil {
		rules = append(rules, formatRCTLRule(id, "maxproc", "deny", *proc.Limit))
	}
	return rules
}

func makeRCTLRules(id string, resources *runtimespec.FreeBSDResources) []string {
	var rules []string
	if resources.Memory != nil {
		rules = append(rules, makeRCTLMemoryRules(id, resources.Memory)...)
	}
	if resources.FSIO != nil {
		rules = append(rules, makeRCTLFSIORules(id, resources.FSIO)...)
	}
	if resources.Shm != nil {
		rules = append(rules, makeRCTLShmRules(id, resources.Shm)...)
	}
	if resources.CPU != nil {
		rules = append(rules, makeRCTLCPURules(id, resources.CPU)...)
	}
	if resources.Process != nil {
		rules = append(rules, makeRCTLProcessRules(id, resources.Process)...)
	}
	return rules
}

func formatRCTLRule(id string, resource string, action string, amount uint64) string {
	return fmt.Sprintf("jail:%v:%v:%v=%v", id, resource, action, amount)
}
