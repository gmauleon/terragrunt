//go:build engine

package test_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gruntwork-io/terragrunt/engine"

	"github.com/gruntwork-io/terragrunt/config"

	"github.com/gruntwork-io/terragrunt/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFixtureLocalEngine          = "fixtures/engine/local-engine"
	testFixtureRemoteEngine         = "fixtures/engine/remote-engine"
	testFixtureOpenTofuEngine       = "fixtures/engine/opentofu-engine"
	testFixtureOpenTofuRunAll       = "fixtures/engine/opentofu-run-all"
	testFixtureOpenTofuLatestRunAll = "fixtures/engine/opentofu-latest-run-all"

	envVarExperimental = "TG_EXPERIMENTAL_ENGINE"
)

var LocalEngineBinaryPath = "terragrunt-iac-engine-opentofu_rpc_" + testEngineVersion() + "_" + runtime.GOOS + "_" + runtime.GOARCH

func TestEngineLocalPlan(t *testing.T) {
	rootPath := setupLocalEngine(t)

	stdout, stderr, err := runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt plan --terragrunt-non-interactive --terragrunt-forward-tf-stdout --terragrunt-working-dir %s --terragrunt-log-level debug", rootPath))
	require.NoError(t, err)

	assert.Contains(t, stderr, LocalEngineBinaryPath+": plugin address")
	assert.Contains(t, stderr, "starting plugin:")
	assert.Contains(t, stderr, "plugin process exited:")
	assert.Contains(t, stdout, "1 to add, 0 to change, 0 to destroy.")
}

func TestEngineLocalApply(t *testing.T) {
	rootPath := setupLocalEngine(t)

	stdout, stderr, err := runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt apply -auto-approve --terragrunt-non-interactive --terragrunt-forward-tf-stdout --terragrunt-working-dir %s", rootPath))
	require.NoError(t, err)

	assert.Contains(t, stderr, LocalEngineBinaryPath+": plugin address")
	assert.Contains(t, stderr, "starting plugin:")
	assert.Contains(t, stderr, "plugin process exited:")
	assert.Contains(t, stdout, "Apply complete! Resources: 1 added, 0 changed, 0 destroyed.")
}

func TestEngineOpentofu(t *testing.T) {
	t.Setenv(envVarExperimental, "1")

	cleanupTerraformFolder(t, testFixtureOpenTofuEngine)
	tmpEnvPath := copyEnvironment(t, testFixtureOpenTofuEngine)
	rootPath := util.JoinPath(tmpEnvPath, testFixtureOpenTofuEngine)

	stdout, stderr, err := runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt apply -auto-approve --terragrunt-non-interactive --terragrunt-forward-tf-stdout --terragrunt-working-dir %s", rootPath))
	require.NoError(t, err)

	assert.Contains(t, stderr, "starting plugin:")
	assert.Contains(t, stderr, "plugin process exited:")
	assert.Contains(t, stdout, "OpenTofu has been successfully initialized")
	assert.Contains(t, stdout, "Apply complete! Resources: 1 added, 0 changed, 0 destroyed.")
}

func TestEngineRunAllOpentofu(t *testing.T) {
	t.Setenv(envVarExperimental, "1")

	cleanupTerraformFolder(t, testFixtureOpenTofuRunAll)
	tmpEnvPath := copyEnvironment(t, testFixtureOpenTofuRunAll)
	rootPath := util.JoinPath(tmpEnvPath, testFixtureOpenTofuRunAll)

	stdout, stderr, err := runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt run-all apply -no-color -auto-approve --terragrunt-non-interactive --terragrunt-forward-tf-stdout --terragrunt-working-dir %s", rootPath))
	require.NoError(t, err)

	assert.Contains(t, stderr, "starting plugin:")
	assert.Contains(t, stderr, "plugin process exited:")
	assert.Contains(t, stdout, "resource \"local_file\" \"test\"")
	assert.Contains(t, stdout, "filename             = \"./test.txt\"\n")
	assert.Contains(t, stdout, "OpenTofu has been successful")
	assert.Contains(t, stdout, "Tofu Shutdown completed")
	assert.Contains(t, stdout, "Apply complete!")
}

func TestEngineRunAllOpentofuCustomPath(t *testing.T) {
	t.Setenv(envVarExperimental, "1")

	cacheDir, rootPath := setupEngineCache(t)

	stdout, stderr, err := runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt run-all apply -no-color -auto-approve --terragrunt-non-interactive --terragrunt-forward-tf-stdout --terragrunt-working-dir %s", rootPath))
	require.NoError(t, err)

	assert.Contains(t, stderr, "starting plugin:")
	assert.Contains(t, stderr, "plugin process exited:")
	assert.Contains(t, stdout, "OpenTofu has been successful")
	assert.Contains(t, stdout, "Tofu Shutdown completed")
	assert.Contains(t, stdout, "Apply complete!")

	// check if cache folder is not empty
	files, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	assert.NotEmpty(t, files)
}

func TestEngineDownloadOverHttp(t *testing.T) {
	t.Setenv(envVarExperimental, "1")

	cleanupTerraformFolder(t, testFixtureRemoteEngine)
	tmpEnvPath := copyEnvironment(t, testFixtureRemoteEngine)
	rootPath := util.JoinPath(tmpEnvPath, testFixtureRemoteEngine)

	platform := runtime.GOOS
	arch := runtime.GOARCH

	copyAndFillMapPlaceholders(t, util.JoinPath(testFixtureRemoteEngine, "terragrunt.hcl"), util.JoinPath(rootPath, config.DefaultTerragruntConfigPath), map[string]string{
		"__hardcoded_url__": fmt.Sprintf("https://github.com/gruntwork-io/terragrunt-engine-opentofu/releases/download/v0.0.4/terragrunt-iac-engine-opentofu_rpc_v0.0.4_%s_%s.zip", platform, arch),
	})

	stdout, stderr, err := runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt apply -auto-approve --terragrunt-non-interactive --terragrunt-forward-tf-stdout --terragrunt-working-dir %s", rootPath))
	require.NoError(t, err)

	assert.Contains(t, stderr, "starting plugin:")
	assert.Contains(t, stderr, "plugin process exited:")
	assert.Contains(t, stdout, "OpenTofu has been successfully initialized")
	assert.Contains(t, stdout, "Apply complete! Resources: 1 added, 0 changed, 0 destroyed.")
}

func TestEngineChecksumVerification(t *testing.T) {
	t.Setenv(envVarExperimental, "1")

	cachePath, rootPath := setupEngineCache(t)

	_, _, err := runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt run-all apply -no-color -auto-approve --terragrunt-non-interactive --terragrunt-working-dir %s", rootPath))
	require.NoError(t, err)

	// change the checksum of the package file
	version := "v0.0.5"
	platform := runtime.GOOS
	arch := runtime.GOARCH
	executablePath := fmt.Sprintf("terragrunt/plugins/iac-engine/rpc/%s/%s/%s/terragrunt-iac-engine-opentofu_rpc_%s_%s_%s", version, platform, arch, version, platform, arch)
	fullPath := util.JoinPath(cachePath, executablePath)

	// open the file and write some data
	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_WRONLY, 0600)
	assert.NoError(t, err)
	nonExecutableData := []byte{0x00}
	if _, err := file.Write(nonExecutableData); err != nil {
		assert.NoError(t, err)
	}

	assert.NoError(t, file.Close())
	_, _, err = runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt run-all apply -no-color -auto-approve --terragrunt-non-interactive --terragrunt-working-dir %s", rootPath))
	require.Error(t, err)

	require.Contains(t, err.Error(), "checksum list has unexpected SHA-256 hash")
}

func TestEngineDisableChecksumCheck(t *testing.T) {
	t.Setenv(envVarExperimental, "1")

	cachePath, rootPath := setupEngineCache(t)

	_, _, err := runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt run-all apply -no-color -auto-approve --terragrunt-non-interactive --terragrunt-working-dir %s", rootPath))
	require.NoError(t, err)

	err = filepath.Walk(cachePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(filepath.Base(path), "_SHA256SUMS") {
			// clean checksum list
			if err := os.Truncate(path, 0); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	// create separated directory for new tests
	cleanupTerraformFolder(t, testFixtureOpenTofuRunAll)
	tmpEnvPath := copyEnvironment(t, testFixtureOpenTofuRunAll)
	rootPath = util.JoinPath(tmpEnvPath, testFixtureOpenTofuRunAll)

	_, _, err = runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt run-all apply -no-color -auto-approve --terragrunt-non-interactive --terragrunt-working-dir %s", rootPath))
	require.Error(t, err)
	require.Contains(t, err.Error(), "verification failure")

	// disable checksum check
	t.Setenv(engine.EngineSkipCheckEnv, "1")

	_, _, err = runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt run-all apply -no-color -auto-approve --terragrunt-non-interactive --terragrunt-working-dir %s", rootPath))
	require.NoError(t, err)
}

func TestEngineOpentofuLatestRunAll(t *testing.T) {
	t.Setenv(envVarExperimental, "1")

	cleanupTerraformFolder(t, testFixtureOpenTofuLatestRunAll)
	tmpEnvPath := copyEnvironment(t, testFixtureOpenTofuLatestRunAll)
	rootPath := util.JoinPath(tmpEnvPath, testFixtureOpenTofuLatestRunAll)

	stdout, _, err := runTerragruntCommandWithOutput(t, fmt.Sprintf("terragrunt run-all apply -no-color -auto-approve --terragrunt-non-interactive --terragrunt-forward-tf-stdout --terragrunt-working-dir %s", rootPath))
	require.NoError(t, err)

	assert.Contains(t, stdout, "resource \"local_file\" \"test\"")
	assert.Contains(t, stdout, "filename             = \"./test.txt\"\n")
	assert.Contains(t, stdout, "Tofu Shutdown completed")
	assert.Contains(t, stdout, "Apply complete!")
}

func setupEngineCache(t *testing.T) (string, string) {
	// create temporary folder
	cacheDir := t.TempDir()
	t.Setenv("TG_ENGINE_CACHE_PATH", cacheDir)

	cleanupTerraformFolder(t, testFixtureOpenTofuRunAll)
	tmpEnvPath := copyEnvironment(t, testFixtureOpenTofuRunAll)
	rootPath := util.JoinPath(tmpEnvPath, testFixtureOpenTofuRunAll)
	return cacheDir, rootPath
}

func setupLocalEngine(t *testing.T) string {
	t.Setenv(envVarExperimental, "1")

	cleanupTerraformFolder(t, testFixtureLocalEngine)
	tmpEnvPath := copyEnvironment(t, testFixtureLocalEngine)
	rootPath := util.JoinPath(tmpEnvPath, testFixtureLocalEngine)

	// get pwd
	pwd, err := os.Getwd()
	require.NoError(t, err)

	copyAndFillMapPlaceholders(t, util.JoinPath(testFixtureLocalEngine, "terragrunt.hcl"), util.JoinPath(rootPath, config.DefaultTerragruntConfigPath), map[string]string{
		"__engine_source__": pwd + "/../" + LocalEngineBinaryPath,
	})
	return rootPath
}

// testEngineVersion returns the version of the engine to be used in the test
func testEngineVersion() string {
	value, found := os.LookupEnv("TOFU_ENGINE_VERSION")
	if !found {
		return "v0.0.1"
	}
	return value
}
