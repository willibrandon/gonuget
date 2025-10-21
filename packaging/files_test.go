package packaging

import (
	"testing"
)

func TestIsRuntimesFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"runtimes file", "runtimes/win-x64/native/library.dll", true},
		{"RUNTIMES file", "RUNTIMES/linux-x64/lib/test.so", true},
		{"lib file", "lib/net6.0/test.dll", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRuntimesFile(tt.path)
			if got != tt.want {
				t.Errorf("IsRuntimesFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsAnalyzerFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"analyzer file", "analyzers/dotnet/cs/MyAnalyzer.dll", true},
		{"ANALYZERS file", "ANALYZERS/roslyn/MyAnalyzer.dll", true},
		{"lib file", "lib/net6.0/test.dll", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAnalyzerFile(tt.path)
			if got != tt.want {
				t.Errorf("IsAnalyzerFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"dll", "lib/net6.0/test.dll", ".dll"},
		{"DLL uppercase", "lib/net6.0/TEST.DLL", ".dll"},
		{"exe", "tools/tool.exe", ".exe"},
		{"nuspec", "package.nuspec", ".nuspec"},
		{"no extension", "README", ""},
		{"multiple dots", "app.config.xml", ".xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFileExtension(tt.path)
			if got != tt.want {
				t.Errorf("GetFileExtension(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsDllOrExe(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"dll", "lib/net6.0/test.dll", true},
		{"DLL uppercase", "lib/net6.0/TEST.DLL", true},
		{"exe", "tools/tool.exe", true},
		{"EXE uppercase", "tools/TOOL.EXE", true},
		{"winmd", "lib/uap10.0/component.winmd", false},
		{"txt", "content/readme.txt", false},
		{"no extension", "README", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDllOrExe(tt.path)
			if got != tt.want {
				t.Errorf("IsDllOrExe(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsAssembly(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"dll", "lib/net6.0/test.dll", true},
		{"DLL uppercase", "lib/net6.0/TEST.DLL", true},
		{"exe", "tools/tool.exe", true},
		{"winmd", "lib/uap10.0/component.winmd", true},
		{"WINMD uppercase", "lib/uap10.0/COMPONENT.WINMD", true},
		{"txt", "content/readme.txt", false},
		{"xml", "build/package.targets", false},
		{"no extension", "README", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAssembly(tt.path)
			if got != tt.want {
				t.Errorf("IsAssembly(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
