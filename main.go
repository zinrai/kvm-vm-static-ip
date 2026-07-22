package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const interfacesDir = "/etc/network/interfaces.d/"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	help := flag.Bool("help", false, "Display help information")
	dryRun := flag.Bool("dry-run", false, "Show the interfaces.d file contents without copying to VM")
	iface := flag.String("interface", "", "Interface name to configure (required)")
	address := flag.String("address", "", "IPv4 address in CIDR notation, e.g. 10.10.0.11/24 (required)")
	gateway := flag.String("gateway", "", "Default gateway address")
	verbose := flag.Bool("verbose", false, "Display verbose output")
	showVersion := flag.Bool("version", false, "Print version information and exit")

	flag.Usage = displayHelp
	flag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	if *help {
		displayHelp()
		return nil
	}

	args := flag.Args()
	if len(args) != 1 {
		displayHelp()
		return fmt.Errorf("VM name is required")
	}
	vmName := args[0]

	if *iface == "" {
		return fmt.Errorf("--interface is required")
	}
	if *address == "" {
		return fmt.Errorf("--address is required")
	}

	// Validate the CIDR address at load time. The parsed result is not used
	// for output (trixie ifupdown accepts CIDR notation directly), but
	// parsing rejects malformed input such as 10.10.0.11/99 before anything
	// is written to the VM.
	if _, _, err := net.ParseCIDR(*address); err != nil {
		return fmt.Errorf("invalid --address %q: %v", *address, err)
	}

	if *gateway != "" {
		if ip := net.ParseIP(*gateway); ip == nil {
			return fmt.Errorf("invalid --gateway %q", *gateway)
		}
	}

	if *verbose {
		fmt.Printf("Processing VM: %s, interface: %s\n", vmName, *iface)
	}

	if err := checkVMStatus(vmName); err != nil {
		return err
	}

	content := generateInterfaceConfig(*iface, *address, *gateway)

	fmt.Printf("Generated %s:\n", *iface)
	fmt.Println("----------------------------------------")
	fmt.Print(content)
	fmt.Println("----------------------------------------")

	if *dryRun {
		fmt.Println("Dry run completed. interfaces.d file not copied to VM.")
		return nil
	}

	if err := copyConfigToVM(*iface, content, vmName, *verbose); err != nil {
		return fmt.Errorf("failed to copy interfaces.d file to VM: %v", err)
	}

	fmt.Printf("Successfully configured static IP for interface '%s' on VM '%s'\n", *iface, vmName)
	fmt.Printf("Start the VM with: sudo virsh start %s\n", vmName)
	return nil
}

func displayHelp() {
	fmt.Println("kvm-vm-static-ip - Assign a static IP to a KVM VM interface via interfaces(5) drop-in")
	fmt.Println("\nUsage:")
	fmt.Println("  kvm-vm-static-ip [flags] <vm-name>")
	fmt.Println("\nFlags:")
	flag.PrintDefaults()
}

// Build an interfaces(5) stanza for a single statically configured interface.
func generateInterfaceConfig(iface, address, gateway string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "auto %s\n", iface)
	fmt.Fprintf(&b, "iface %s inet static\n", iface)
	fmt.Fprintf(&b, "    address %s\n", address)
	if gateway != "" {
		fmt.Fprintf(&b, "    gateway %s\n", gateway)
	}
	return b.String()
}

// Check if VM exists and is shut off.
func checkVMStatus(vmName string) error {
	cmd := exec.Command("sudo", "virsh", "list", "--state-shutoff", "--name")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute virsh command: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == vmName {
			return nil
		}
	}

	cmd = exec.Command("sudo", "virsh", "list", "--all", "--name")
	output, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute virsh command: %v", err)
	}

	scanner = bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == vmName {
			return fmt.Errorf("VM '%s' exists but is currently running. Please shut it down first", vmName)
		}
	}

	return fmt.Errorf("VM '%s' does not exist", vmName)
}

// Write the config to a temp file named after the interface and copy it into
// the VM at /etc/network/interfaces.d/ with virt-copy-in.
func copyConfigToVM(iface, content, vmName string, verbose bool) error {
	tempPath := filepath.Join(os.TempDir(), iface)
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create temporary file %s: %v", tempPath, err)
	}
	defer os.Remove(tempPath)

	if verbose {
		fmt.Printf("Copying %s to %s on VM '%s'\n", iface, interfacesDir, vmName)
	}

	cmd := exec.Command("sudo", "virt-copy-in", "-d", vmName, tempPath, interfacesDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy %s: %v\nOutput: %s", iface, err, out.String())
	}

	return nil
}
