package gonix

import (
	"fmt"
	"runtime"
)

// OS identifies the operating-system component of a Nix system.
type OS string

const (
	// OSAIX identifies IBM AIX.
	OSAIX OS = "aix"
	// OSAndroid identifies Android.
	OSAndroid OS = "android"
	// OSCygwin identifies Cygwin.
	OSCygwin OS = "cygwin"
	// OSDarwin identifies Darwin-based systems.
	OSDarwin OS = "darwin"
	// OSDragonFly identifies DragonFly BSD.
	OSDragonFly OS = "dragonfly"
	// OSFreeBSD identifies FreeBSD.
	OSFreeBSD OS = "freebsd"
	// OSGenode identifies Genode.
	OSGenode OS = "genode"
	// OSGHCJS identifies the GHCJS JavaScript target.
	OSGHCJS OS = "ghcjs"
	// OSHurd identifies GNU Hurd.
	OSHurd OS = "hurd"
	// OSIllumos identifies illumos.
	OSIllumos OS = "illumos"
	// OSIOS identifies iOS.
	OSIOS OS = "ios"
	// OSLinux identifies Linux.
	OSLinux OS = "linux"
	// OSMMIXWare identifies MMIXware.
	OSMMIXWare OS = "mmixware"
	// OSNetBSD identifies NetBSD.
	OSNetBSD OS = "netbsd"
	// OSNone identifies a bare-metal or freestanding target.
	OSNone OS = "none"
	// OSOpenBSD identifies OpenBSD.
	OSOpenBSD OS = "openbsd"
	// OSPlan9 identifies Plan 9.
	OSPlan9 OS = "plan9"
	// OSRedox identifies Redox.
	OSRedox OS = "redox"
	// OSSolaris identifies Solaris.
	OSSolaris OS = "solaris"
	// OSUEFI identifies UEFI.
	OSUEFI OS = "uefi"
	// OSWasi identifies WASI.
	OSWasi OS = "wasi"
	// OSWasip1 identifies Go's WASI preview 1 target.
	OSWasip1 OS = "wasip1"
	// OSWindows identifies Windows.
	OSWindows OS = "windows"
)

// String returns the Nix operating-system identifier.
func (os OS) String() string {
	return string(os)
}

// Arch identifies the architecture component of a Nix system.
type Arch string

const (
	// Arch386 identifies Go's 386 architecture as Nix i686.
	Arch386 Arch = ArchI686
	// ArchAarch64 identifies 64-bit ARM.
	ArchAarch64 Arch = "aarch64"
	// ArchAarch64BE identifies big-endian 64-bit ARM.
	ArchAarch64BE Arch = "aarch64_be"
	// ArchArc identifies Synopsys ARC.
	ArchArc Arch = "arc"
	// ArchArm identifies generic ARM.
	ArchArm Arch = "arm"
	// ArchArmv5tel identifies ARMv5 little-endian.
	ArchArmv5tel Arch = "armv5tel"
	// ArchArmv6l identifies ARMv6 little-endian.
	ArchArmv6l Arch = "armv6l"
	// ArchArmv7a identifies ARMv7-A.
	ArchArmv7a Arch = "armv7a"
	// ArchArmv7l identifies ARMv7 little-endian.
	ArchArmv7l Arch = "armv7l"
	// ArchAvr identifies AVR.
	ArchAvr Arch = "avr"
	// ArchI686 identifies 32-bit x86.
	ArchI686 Arch = "i686"
	// ArchJavascript identifies JavaScript.
	ArchJavascript Arch = "javascript"
	// ArchLoong64 identifies Go's loong64 architecture as Nix loongarch64.
	ArchLoong64 Arch = ArchLoongArch64
	// ArchLoongArch64 identifies LoongArch64.
	ArchLoongArch64 Arch = "loongarch64"
	// ArchM68k identifies Motorola 68000.
	ArchM68k Arch = "m68k"
	// ArchMicroBlaze identifies MicroBlaze.
	ArchMicroBlaze Arch = "microblaze"
	// ArchMicroBlazeEL identifies little-endian MicroBlaze.
	ArchMicroBlazeEL Arch = "microblazeel"
	// ArchMIPS identifies big-endian MIPS.
	ArchMIPS Arch = "mips"
	// ArchMIPS64 identifies big-endian MIPS64.
	ArchMIPS64 Arch = "mips64"
	// ArchMIPS64EL identifies little-endian MIPS64.
	ArchMIPS64EL Arch = "mips64el"
	// ArchMIPSEL identifies little-endian MIPS.
	ArchMIPSEL Arch = "mipsel"
	// ArchMIPSLE identifies Go's mipsle architecture as Nix mipsel.
	ArchMIPSLE Arch = ArchMIPSEL
	// ArchMMIX identifies MMIX.
	ArchMMIX Arch = "mmix"
	// ArchMSP430 identifies MSP430.
	ArchMSP430 Arch = "msp430"
	// ArchOR1K identifies OpenRISC 1000.
	ArchOR1K Arch = "or1k"
	// ArchPPC64 identifies Go's ppc64 architecture as Nix powerpc64.
	ArchPPC64 Arch = ArchPowerPC64
	// ArchPPC64LE identifies Go's ppc64le architecture as Nix powerpc64le.
	ArchPPC64LE Arch = ArchPowerPC64LE
	// ArchPowerPC identifies PowerPC.
	ArchPowerPC Arch = "powerpc"
	// ArchPowerPC64 identifies 64-bit PowerPC.
	ArchPowerPC64 Arch = "powerpc64"
	// ArchPowerPC64LE identifies little-endian 64-bit PowerPC.
	ArchPowerPC64LE Arch = "powerpc64le"
	// ArchPowerPCLE identifies little-endian PowerPC.
	ArchPowerPCLE Arch = "powerpcle"
	// ArchRISCV32 identifies 32-bit RISC-V.
	ArchRISCV32 Arch = "riscv32"
	// ArchRISCV64 identifies 64-bit RISC-V.
	ArchRISCV64 Arch = "riscv64"
	// ArchRX identifies Renesas RX.
	ArchRX Arch = "rx"
	// ArchS390 identifies IBM S/390.
	ArchS390 Arch = "s390"
	// ArchS390X identifies IBM z/Architecture.
	ArchS390X Arch = "s390x"
	// ArchSH4 identifies SuperH SH-4.
	ArchSH4 Arch = "sh4"
	// ArchVC4 identifies VideoCore IV.
	ArchVC4 Arch = "vc4"
	// ArchWasm32 identifies 32-bit WebAssembly.
	ArchWasm32 Arch = "wasm32"
	// ArchWasm64 identifies 64-bit WebAssembly.
	ArchWasm64 Arch = "wasm64"
	// ArchX86_64 identifies 64-bit x86.
	ArchX86_64 Arch = "x86_64"
)

// String returns the Nix architecture identifier.
func (a Arch) String() string {
	return string(a)
}

// DefaultSystem returns the Nix system identifier for the current Go runtime.
func DefaultSystem() string {
	// nixpkgs system identifier for the JavaScript target is historically javascript-ghcjs, not wasm32-js
	if runtime.GOOS == "js" && runtime.GOARCH == "wasm" {
		return fmt.Sprintf("%s-%s", ArchJavascript, OSGHCJS)
	}

	return fmt.Sprintf("%s-%s", getArch(), getOS())
}

func getOS() OS {
	switch runtime.GOOS {
	case "aix":
		return OSAIX
	case "android":
		return OSAndroid
	case "darwin":
		return OSDarwin
	case "dragonfly":
		return OSDragonFly
	case "freebsd":
		return OSFreeBSD
	case "hurd":
		return OSHurd
	case "illumos":
		return OSIllumos
	case "ios":
		return OSIOS
	case "linux":
		return OSLinux
	case "netbsd":
		return OSNetBSD
	case "openbsd":
		return OSOpenBSD
	case "plan9":
		return OSPlan9
	case "solaris":
		return OSSolaris
	case "wasip1":
		return OSWasi
	case "windows":
		return OSWindows
	default:
		return OS(runtime.GOOS)
	}
}

func getArch() Arch {
	switch runtime.GOARCH {
	case "386":
		return ArchI686
	case "amd64":
		return ArchX86_64
	case "arm":
		return ArchArm
	case "arm64":
		return ArchAarch64
	case "loong64":
		return ArchLoongArch64
	case "mips":
		return ArchMIPS
	case "mipsle":
		return ArchMIPSEL
	case "mips64":
		return ArchMIPS64
	case "mips64le":
		return ArchMIPS64EL
	case "ppc64":
		return ArchPowerPC64
	case "ppc64le":
		return ArchPowerPC64LE
	case "riscv64":
		return ArchRISCV64
	case "s390x":
		return ArchS390X
	case "wasm":
		return ArchWasm32
	default:
		return Arch(runtime.GOARCH)
	}
}
