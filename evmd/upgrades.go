package evmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
)

// UpgradeName defines the on-chain upgrade name for the sample EVMD upgrade
// from v0.4.0 to v0.5.0.
//
// NOTE: This upgrade defines a reference implementation of what an upgrade
// could look like when an application is migrating from EVMD version
// v0.4.0 to v0.5.x
const UpgradeName = "v0.5.0-to-v0.6.0"

func (app EVMD) RegisterUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Debug("this is a debug level message to test that verbose logging mode has properly been enabled during a chain upgrade")
			return app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
		},
	)

	upgradeInfo, err := readUpgradeInfoFromDiskIfPresent(app.homePath)
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

// readUpgradeInfoFromDiskIfPresent mirrors x/upgrade's read path without
// creating the data directory on empty reads.
func readUpgradeInfoFromDiskIfPresent(homePath string) (upgradetypes.Plan, error) {
	var upgradeInfo upgradetypes.Plan

	upgradeInfoPath := filepath.Join(homePath, "data", upgradetypes.UpgradeInfoFilename)
	data, err := os.ReadFile(upgradeInfoPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return upgradeInfo, nil
		}

		return upgradeInfo, err
	}

	if err := json.Unmarshal(data, &upgradeInfo); err != nil {
		return upgradeInfo, err
	}

	return upgradeInfo, nil
}
