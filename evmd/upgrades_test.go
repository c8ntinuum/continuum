package evmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	upgradetypes "cosmossdk.io/x/upgrade/types"
)

func TestReadUpgradeInfoFromDiskIfPresentDoesNotCreateRelativeDataDir(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(cwd))
	})

	upgradeInfo, err := readUpgradeInfoFromDiskIfPresent("")
	require.NoError(t, err)
	require.Equal(t, upgradetypes.Plan{}, upgradeInfo)

	_, err = os.Stat(filepath.Join(tmpDir, "data"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestReadUpgradeInfoFromDiskIfPresentReadsExistingUpgradeInfo(t *testing.T) {
	homePath := t.TempDir()
	dataDir := filepath.Join(homePath, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))

	expected := upgradetypes.Plan{
		Name:   "test-upgrade",
		Height: 42,
		Info:   "details",
	}

	bz, err := json.Marshal(expected)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, upgradetypes.UpgradeInfoFilename), bz, 0o600))

	upgradeInfo, err := readUpgradeInfoFromDiskIfPresent(homePath)
	require.NoError(t, err)
	require.Equal(t, expected, upgradeInfo)
}
