package main

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const minAndroidAPI = 15

// TODO 可以手动修改？
var buildAndroidAPI = minAndroidAPI

var androidEnv map[string][]string // android arch -> []string

type ndkToolchain struct {
	arch        string
	abi         string
	minAPI      int
	toolPrefix  string
	clangPrefix string
}

func (tc *ndkToolchain) ClangPrefix() string {
	if buildAndroidAPI < tc.minAPI {
		return fmt.Sprintf("%s%d", tc.clangPrefix, tc.minAPI)
	}
	return fmt.Sprintf("%s%d", tc.clangPrefix, buildAndroidAPI)
}

func archNDK() string {
	if runtime.GOOS == "windows" && runtime.GOARCH == "386" {
		return "windows"
	} else {
		var arch string
		switch runtime.GOARCH {
		case "386":
			arch = "x86"
		case "amd64":
			arch = "x86_64"
		default:
			panic("unsupported GOARCH: " + runtime.GOARCH)
		}
		return runtime.GOOS + "-" + arch
	}
}

func (tc *ndkToolchain) Path(ndkRoot, toolName string) string {
	var pref string
	switch toolName {
	case "clang", "clang++":
		pref = tc.ClangPrefix()
	default:
		pref = tc.toolPrefix
	}
	return filepath.Join(ndkRoot, "toolchains", "llvm", "prebuilt", archNDK(), "bin", pref+"-"+toolName)
}

type ndkConfig map[string]ndkToolchain // map: GOOS->androidConfig.

func (nc ndkConfig) Toolchain(arch string) ndkToolchain {
	tc, ok := nc[arch]
	if !ok {
		panic(`unsupported architecture: ` + arch)
	}
	return tc
}

var ndk = ndkConfig{
	"arm": {
		arch:        "arm",
		abi:         "armeabi-v7a",
		minAPI:      16,
		toolPrefix:  "arm-linux-androideabi",
		clangPrefix: "armv7a-linux-androideabi",
	},
	"arm64": {
		arch:        "arm64",
		abi:         "arm64-v8a",
		minAPI:      21,
		toolPrefix:  "aarch64-linux-android",
		clangPrefix: "aarch64-linux-android",
	},
	"386": {
		arch:        "x86",
		abi:         "x86",
		minAPI:      16,
		toolPrefix:  "i686-linux-android",
		clangPrefix: "i686-linux-android",
	},
	"amd64": {
		arch:        "x86_64",
		abi:         "x86_64",
		minAPI:      21,
		toolPrefix:  "x86_64-linux-android",
		clangPrefix: "x86_64-linux-android",
	},
}

func compareVersion(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	var pre1, pre2 string
	var post1, post2 string
	if index1 := strings.Index(s1, "."); index1 == -1 {
		pre1 = s1
	} else {
		pre1 = s1[:index1]
		post1 = s1[index1+1:]
	}
	if index2 := strings.Index(s2, "."); index2 == -1 {
		pre2 = s2
	} else {
		pre2 = s2[:index2]
		post2 = s2[index2+1:]
	}
	var i1, i2 int
	i1, _ = strconv.Atoi(pre1)
	i2, _ = strconv.Atoi(pre2)
	if i1 == i2 {
		return compareVersion(post1, post2)
	} else if i1 > i2 {
		return 1
	} else {
		return -1
	}
}

func ndkRoot() (string, error) {
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome != "" {
		ndkRoot := filepath.Join(androidHome, "ndk-bundle")
		_, err := os.Stat(ndkRoot)
		if err == nil {
			return ndkRoot, nil
		}

		ndkRoot = filepath.Join(androidHome, "ndk")
		dir, _ := os.Open(ndkRoot)
		if dir != nil {
			infos, _ := dir.Readdir(-1)
			var max string
			for _, info := range infos {
				if compareVersion(max, info.Name()) < 0 {
					max = info.Name()
				}
			}
			if len(max) > 0 {
				return filepath.Join(ndkRoot, max), nil
			}
		}
	}

	ndkPaths := []string{"NDK", "NDK_HOME", "NDK_ROOT", "ANDROID_NDK_HOME"}
	ndkRoot := ""
	for _, path := range ndkPaths {
		ndkRoot = os.Getenv(path)
		if ndkRoot != "" {
			_, err := os.Stat(ndkRoot)
			if err == nil {
				return ndkRoot, nil
			}
		}
	}

	return "", fmt.Errorf("no Android NDK found in $ANDROID_HOME/ndk-bundle, $ANDROID_HOME/ndk, $NDK_HOME, $NDK_ROOT nor in $ANDROID_NDK_HOME")
}

func arch(f string) (string, error) {
	elfFile, err := elf.Open(f)
	if err != nil {
		return "", err
	}
	defer elfFile.Close()

	switch elfFile.Machine {
	case elf.EM_ARM:
		return "arm", nil
	case elf.EM_AARCH64:
		return "arm64", nil
	case elf.EM_X86_64:
		return "amd64", nil
	case elf.EM_386:
		return "386", nil
	default:
		return elfFile.Machine.String(), nil
	}
}

func run(exe string, args ...string) error {
	command := exec.Command(exe, args...)
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout
	return command.Run()
}

func stripPath(a string) (string, error) {
	ndkRoot, err := ndkRoot()
	if err != nil {
		return "", err
	}

	if toolchain, ok := ndk[a]; ok {
		return toolchain.Path(ndkRoot, "strip"), nil
	}
	return "", fmt.Errorf("unknown arch: %q", a)
}
