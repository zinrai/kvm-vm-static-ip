# kvm-vm-static-ip

A command-line tool to assign a static IP address to a single KVM virtual machine interface.

## What it does

Generates an interfaces(5) drop-in for one network interface and copies it into the VM at `/etc/network/interfaces.d/<interface>` with `virt-copy-in`. The drop-in defines the interface as `inet static` with the given address, and optionally a gateway and DNS nameservers. This relies on the default `source /etc/network/interfaces.d/*` line present in a stock Debian `/etc/network/interfaces`.

The interface is configured one at a time. To assign addresses to multiple interfaces or multiple VMs, run the tool once per interface, for example in a shell loop.

The VM must be in the shutoff state. The tool refuses to run against a running VM.

## Prerequisites

- KVM/QEMU virtualization environment
- A guest using ifupdown with `source /etc/network/interfaces.d/*` in `/etc/network/interfaces`
- `libvirt-clients` package (for `virsh`)
- `libguestfs-tools` package (for `virt-copy-in`)
- Sudo privileges

## Usage

```
$ kvm-vm-static-ip [flags] <vm-name>
```

Flags:

- `--help`: Display help information
- `--dry-run`: Show the drop-in file contents without copying to the VM
- `--interface`: Interface name to configure (required)
- `--address`: IPv4 address in CIDR notation, e.g. 192.0.2.11/24 (required)
- `--gateway`: Default gateway address
- `--verbose`: Display verbose output

The drop-in file is named after the interface, so `--interface net0` writes to `/etc/network/interfaces.d/net0`.

### Examples

Assign an address with a gateway:

```bash
$ kvm-vm-static-ip --interface net0 --address 192.0.2.11/24 --gateway 192.0.2.1 debian-vm
```

This generates `/etc/network/interfaces.d/net0`:

```
auto net0
iface net0 inet static
    address 192.0.2.11/24
    gateway 192.0.2.1
```

Assign an address only:

```bash
$ kvm-vm-static-ip --interface net1 --address 198.51.100.11/24 debian-vm
```

Preview without copying:

```bash
$ kvm-vm-static-ip --dry-run --interface net0 --address 192.0.2.11/24 debian-vm
```

Configure several interfaces on one VM:

```bash
$ kvm-vm-static-ip --interface net0 --address 192.0.2.11/24 --gateway 192.0.2.1 debian-vm
$ kvm-vm-static-ip --interface net1 --address 198.51.100.11/24 debian-vm
```

## DNS

This tool does not configure DNS. The `dns-nameservers` directive in interfaces(5) requires the resolvconf package in the guest, so DNS is left out to avoid that dependency. Configure DNS separately, for example by writing `/etc/resolv.conf` into the VM with a file injection tool.

## License

This project is licensed under the [MIT License](./LICENSE).
