package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
)

func artifact(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(stdout, "Usage: bridge artifact <inspect|validate|evaluate> <artifact.zip>")
		return 0
	}
	if len(args) < 2 {
		fmt.Fprintln(stderr, "error: artifact path is required")
		return exitUsage
	}
	switch args[0] {
	case "inspect":
		return artifactInspect(args[1], stdout, stderr)
	case "validate":
		return artifactValidate(args[1], stdout, stderr)
	case "evaluate":
		return artifactEvaluate(args[1], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "error: unsupported artifact subcommand", args[0])
		return exitUsage
	}
}

func artifactInspect(path string, stdout, stderr io.Writer) int {
	zr, err := zip.OpenReader(path)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	defer zr.Close()
	fmt.Fprintf(stdout, "Artifact: %s\nFiles: %d\n", filepath.Base(path), len(zr.File))
	for _, f := range zr.File {
		fmt.Fprintln(stdout, f.Name)
	}
	return 0
}

func artifactValidate(path string, stdout, stderr io.Writer) int {
	if err := validateArtifactArchive(path); err != nil {
		fmt.Fprintln(stderr, "invalid:", err)
		return exitUsage
	}
	fmt.Fprintln(stdout, "valid:", path)
	return 0
}

func validateArtifactArchive(path string) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer zr.Close()
	if len(zr.File) == 0 {
		return fmt.Errorf("empty artifact")
	}

	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		clean := filepath.ToSlash(filepath.Clean(f.Name))
		if clean == "." || strings.HasPrefix(clean, "../") || filepath.IsAbs(f.Name) || strings.HasPrefix(f.Name, "/") {
			return fmt.Errorf("unsafe archive path %q", f.Name)
		}
		if _, exists := files[clean]; exists {
			return fmt.Errorf("duplicate archive entry %q", clean)
		}
		files[clean] = f
	}

	required := []string{"manifest.json", "scenario.json", "runs.jsonl", "result.json", "healthy.json", "environment.json", "summary.csv", "handoffs.csv"}
	for _, name := range required {
		if files[name] == nil {
			return fmt.Errorf("required file %s is missing", name)
		}
	}

	var scenario traffic.BenchmarkScenario
	if err := readZipJSON(files["scenario.json"], &scenario); err != nil {
		return fmt.Errorf("scenario.json: %w", err)
	}
	if scenario.SchemaVersion != traffic.BenchmarkSchemaV1 {
		return fmt.Errorf("scenario.json: schema_version must be %q", traffic.BenchmarkSchemaV1)
	}
	if err := scenario.Validate(); err != nil {
		return fmt.Errorf("scenario.json: %w", err)
	}

	var result traffic.BenchmarkResult
	if err := readZipJSON(files["result.json"], &result); err != nil {
		return fmt.Errorf("result.json: %w", err)
	}
	if result.SchemaVersion != traffic.BenchmarkResultSchemaV1 {
		return fmt.Errorf("result.json: schema_version must be %q", traffic.BenchmarkResultSchemaV1)
	}
	if result.ExecutionID == "" || result.SuiteID == "" {
		return fmt.Errorf("result.json: execution_id and suite_id are required")
	}
	if result.SuiteID != scenario.Suite.ID {
		return fmt.Errorf("suite_id mismatch: result=%q scenario=%q", result.SuiteID, scenario.Suite.ID)
	}

	var manifest struct {
		SchemaVersion      string                          `json:"schema_version"`
		ExecutionID        string                          `json:"execution_id"`
		SuiteID            string                          `json:"suite_id"`
		IntegrityAlgorithm string                          `json:"integrity_algorithm"`
		ArtifactFiles      []traffic.ArtifactFileIntegrity `json:"artifact_files"`
	}
	if err := readZipJSON(files["manifest.json"], &manifest); err != nil {
		return fmt.Errorf("manifest.json: %w", err)
	}
	if manifest.SchemaVersion != "bridge.benchmark.execution.v1" {
		return fmt.Errorf("manifest.json: unsupported schema_version %q", manifest.SchemaVersion)
	}
	if manifest.ExecutionID != result.ExecutionID || manifest.SuiteID != result.SuiteID {
		return fmt.Errorf("manifest identity does not match result.json")
	}
	if manifest.IntegrityAlgorithm != "sha256" {
		return fmt.Errorf("manifest.json: integrity_algorithm must be sha256")
	}
	listed := make(map[string]struct{}, len(manifest.ArtifactFiles))
	for _, entry := range manifest.ArtifactFiles {
		clean := filepath.ToSlash(filepath.Clean(entry.Path))
		if clean == "." || clean == "manifest.json" || strings.HasPrefix(clean, "../") || filepath.IsAbs(entry.Path) {
			return fmt.Errorf("manifest.json: invalid artifact file path %q", entry.Path)
		}
		if _, exists := listed[clean]; exists {
			return fmt.Errorf("manifest.json: duplicate artifact file %q", clean)
		}
		listed[clean] = struct{}{}
		f := files[clean]
		if f == nil || f.FileInfo().IsDir() {
			return fmt.Errorf("manifest.json: listed file %s is missing", clean)
		}
		if int64(f.UncompressedSize64) != entry.Size {
			return fmt.Errorf("artifact size mismatch for %s: manifest=%d archive=%d", clean, entry.Size, f.UncompressedSize64)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		h := sha256.New()
		_, copyErr := io.Copy(h, io.LimitReader(rc, 512<<20))
		closeErr := rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		actual := hex.EncodeToString(h.Sum(nil))
		if actual != entry.SHA256 {
			return fmt.Errorf("artifact digest mismatch for %s", clean)
		}
	}
	for name, f := range files {
		if f.FileInfo().IsDir() || name == "manifest.json" || strings.HasSuffix(strings.ToLower(name), ".zip") {
			continue
		}
		if _, ok := listed[name]; !ok {
			return fmt.Errorf("unlisted artifact file %s", name)
		}
	}

	runCount, err := countJSONLines(files["runs.jsonl"])
	if err != nil {
		return fmt.Errorf("runs.jsonl: %w", err)
	}
	if runCount != len(result.Runs) {
		return fmt.Errorf("run count mismatch: runs.jsonl=%d result.json=%d", runCount, len(result.Runs))
	}
	return nil
}

func readZipJSON(file *zip.File, dst any) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	dec := json.NewDecoder(io.LimitReader(rc, 128<<20))
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("unexpected trailing JSON data")
	}
	return nil
}

func countJSONLines(file *zip.File) (int, error) {
	rc, err := file.Open()
	if err != nil {
		return 0, err
	}
	defer rc.Close()
	scanner := bufio.NewScanner(io.LimitReader(rc, 256<<20))
	scanner.Buffer(make([]byte, 64<<10), 8<<20)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var run traffic.BenchmarkRun
		if err := json.Unmarshal([]byte(line), &run); err != nil {
			return 0, fmt.Errorf("line %d: %w", count+1, err)
		}
		if run.RunMetadata.RunID == "" {
			return 0, fmt.Errorf("line %d: run_id is required", count+1)
		}
		count++
	}
	return count, scanner.Err()
}

func artifactEvaluate(path string, stdout, stderr io.Writer) int {
	if !strings.HasSuffix(strings.ToLower(path), ".zip") {
		return evaluateBenchmarkResult([]string{"check", path}, stdout, stderr)
	}
	if err := validateArtifactArchive(path); err != nil {
		fmt.Fprintln(stderr, "error: invalid artifact:", err)
		return exitUsage
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	defer zr.Close()
	var resultFile *zip.File
	for _, f := range zr.File {
		if filepath.ToSlash(f.Name) == "result.json" {
			resultFile = f
			break
		}
	}
	rc, err := resultFile.Open()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	data, err := io.ReadAll(io.LimitReader(rc, 128<<20))
	closeErr := rc.Close()
	if err != nil || closeErr != nil {
		if err == nil {
			err = closeErr
		}
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	tmp, err := os.CreateTemp("", "bridge-artifact-result-*.json")
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	if err = tmp.Close(); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	return evaluateBenchmarkResult([]string{"check", name}, stdout, stderr)
}
