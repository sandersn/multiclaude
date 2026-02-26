package prompt

import (
"os"
"path/filepath"
"strings"
"testing"
)

func TestNewBuilder(t *testing.T) {
b := NewBuilder()
if b == nil {
t.Fatal("NewBuilder() returned nil")
}
if b.Len() != 0 {
t.Errorf("expected empty builder, got %d sections", b.Len())
}
}

func TestBuilderAddSection(t *testing.T) {
b := NewBuilder()
b.AddSection("Role", "You are a helpful assistant.")
b.AddSection("Guidelines", "Write clean code.")
if b.Len() != 2 {
t.Errorf("expected 2 sections, got %d", b.Len())
}
result := b.Build()
if !strings.Contains(result, "## Role") {
t.Error("expected '## Role'")
}
if !strings.Contains(result, "## Guidelines") {
t.Error("expected '## Guidelines'")
}
}

func TestBuilderAddSectionEmpty(t *testing.T) {
b := NewBuilder()
b.AddSection("Empty", "")
b.AddSection("Role", "Content")
if b.Len() != 1 {
t.Errorf("expected 1 section (empty skipped), got %d", b.Len())
}
}

func TestBuilderAddRaw(t *testing.T) {
b := NewBuilder()
b.AddRaw("Raw content without header")
result := b.Build()
if !strings.Contains(result, "Raw content without header") {
t.Error("expected raw content")
}
if strings.Contains(result, "## ") {
t.Error("raw content should not have a header")
}
}

func TestBuilderAddRawEmpty(t *testing.T) {
b := NewBuilder()
b.AddRaw("")
if b.Len() != 0 {
t.Errorf("expected 0 sections, got %d", b.Len())
}
}

func TestBuilderBuild(t *testing.T) {
b := NewBuilder()
b.AddSection("First", "First content")
b.AddSection("Second", "Second content")
result := b.Build()
if !strings.Contains(result, "---") {
t.Error("expected sections separated by ---")
}
if strings.Index(result, "First content") > strings.Index(result, "Second content") {
t.Error("expected first before second")
}
}

func TestBuilderBuildEmpty(t *testing.T) {
if NewBuilder().Build() != "" {
t.Error("expected empty string from empty builder")
}
}

func TestBuilderChaining(t *testing.T) {
result := NewBuilder().
AddSection("A", "Content A").
AddSection("B", "Content B").
AddRaw("Raw").
Build()
for _, s := range []string{"Content A", "Content B", "Raw"} {
if !strings.Contains(result, s) {
t.Errorf("expected %q in result", s)
}
}
}

func TestBuilderClear(t *testing.T) {
b := NewBuilder()
b.AddSection("Test", "Content")
b.Clear()
if b.Len() != 0 {
t.Errorf("expected 0 after clear, got %d", b.Len())
}
}

func TestNewLoader(t *testing.T) {
l := NewLoader()
if l == nil {
t.Fatal("NewLoader() returned nil")
}
if l.DefaultPrompts == nil {
t.Error("DefaultPrompts should be initialized")
}
}

func TestLoaderLoad(t *testing.T) {
l := NewLoader()
l.SetDefault(TypeSupervisor, "You are a supervisor.")
result, err := l.Load(TypeSupervisor)
if err != nil {
t.Fatalf("Load() failed: %v", err)
}
if !strings.Contains(result, "You are a supervisor.") {
t.Error("expected default prompt")
}
}

func TestLoaderLoadWithCustomPrompt(t *testing.T) {
tmpDir, err := os.MkdirTemp("", "prompt-test")
if err != nil {
t.Fatal(err)
}
defer os.RemoveAll(tmpDir)

os.WriteFile(filepath.Join(tmpDir, "SUPERVISOR.md"), []byte("Custom instructions."), 0644)

l := NewLoader()
l.SetDefault(TypeSupervisor, "Default prompt")
l.SetCustomDir(tmpDir)

result, err := l.Load(TypeSupervisor)
if err != nil {
t.Fatal(err)
}
if !strings.Contains(result, "Default prompt") {
t.Error("expected default prompt")
}
if !strings.Contains(result, "Custom instructions.") {
t.Error("expected custom prompt")
}
}

func TestLoaderLoadCustomMissing(t *testing.T) {
tmpDir, _ := os.MkdirTemp("", "prompt-test")
defer os.RemoveAll(tmpDir)

l := NewLoader()
l.SetCustomDir(tmpDir)
result, err := l.LoadCustom(TypeSupervisor)
if err != nil {
t.Fatal(err)
}
if result != "" {
t.Errorf("expected empty for missing file, got %q", result)
}
}

func TestLoaderLoadCustomUnknownType(t *testing.T) {
l := NewLoader()
l.SetCustomDir("/tmp")
_, err := l.LoadCustom(AgentType("unknown"))
if err == nil {
t.Error("expected error for unknown type")
}
}

func TestLoaderLoadWithExtras(t *testing.T) {
l := NewLoader()
l.SetDefault(TypeWorker, "Default worker prompt")
extras := map[string]string{"CLI Documentation": "Available commands: ..."}
result, err := l.LoadWithExtras(TypeWorker, extras)
if err != nil {
t.Fatal(err)
}
if !strings.Contains(result, "Default worker prompt") {
t.Error("expected default prompt")
}
if !strings.Contains(result, "CLI Documentation") {
t.Error("expected extras header")
}
}

func TestCustomPromptFilename(t *testing.T) {
tests := []struct {
agentType AgentType
expected  string
}{
{TypeSupervisor, "SUPERVISOR.md"},
{TypeWorker, "WORKER.md"},
{TypeMergeQueue, "REVIEWER.md"},
{TypeWorkspace, "WORKSPACE.md"},
{TypeReview, "REVIEW.md"},
{AgentType("unknown"), ""},
}
for _, tc := range tests {
if got := customPromptFilename(tc.agentType); got != tc.expected {
t.Errorf("%s: expected %q, got %q", tc.agentType, tc.expected, got)
}
}
}

func TestWriteToFile(t *testing.T) {
tmpDir, _ := os.MkdirTemp("", "prompt-test")
defer os.RemoveAll(tmpDir)

path := filepath.Join(tmpDir, "nested", "dir", "prompt.md")
if err := WriteToFile(path, "Test content"); err != nil {
t.Fatal(err)
}
data, _ := os.ReadFile(path)
if string(data) != "Test content" {
t.Errorf("expected 'Test content', got %q", string(data))
}
}

func TestWriteToFileInvalidPath(t *testing.T) {
if err := WriteToFile("/dev/null/subdir/prompt.md", "content"); err == nil {
t.Error("expected error for invalid path")
}
}

func TestLoaderChaining(t *testing.T) {
l := NewLoader().SetDefault(TypeSupervisor, "Default").SetCustomDir("/tmp")
if l.DefaultPrompts[TypeSupervisor] != "Default" {
t.Error("chained SetDefault failed")
}
if l.CustomPromptDir != "/tmp" {
t.Error("chained SetCustomDir failed")
}
}
