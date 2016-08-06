package contractmanager

import (
	"os"
	"path/filepath"

	"github.com/NebulousLabs/Sia/build"
	"github.com/NebulousLabs/Sia/crypto"
	"github.com/NebulousLabs/Sia/persist"
)

type (
	// savedSettings contains all of the contract manager persistent details
	// that are single fields or otherwise do not need to go into a database or
	// onto another disk.
	savedSettings struct {
		SectorSalt     crypto.Hash
		StorageFolders []*storageFolder
	}

	// sectorLocationEntry contains all the information necessary to add an
	// entry to the sectorLocations map in the contract manager.
	sectorLocationEntry struct {
		index         uint32
		sectorID      string
		storageFolder uint16
	}
)

// initSettings will set the default settings for the contract manager.
// initSettings should only be run for brand new contract maangers.
func (cm *ContractManager) initSettings() error {
	// Typically any time a change is made to the persistent state of the
	// contract manager, the write ahead log should be used. We have a unique
	// situation where there is a brand new contract manager, and we can rely
	// on the safety features of persist.SaveFileSync to be certain that
	// everything will be saved cleanly to disk or not at all before the
	// function is returned, therefore the WAL is not used, and this saves some
	// code, especially regarding changes to the sector salt. Aside from
	// initialization, the sector salt is never changed.

	// Initialize the sector salt to a random value.
	_, err := cm.dependencies.randRead(cm.sectorSalt[:])
	if err != nil {
		return build.ExtendErr("error creating salt for contract manager", err)
	}

	// Ensure that the initialized defaults have stuck by doing a SaveFileSync
	// with the new settings values.
	return build.ExtendErr("error saving contract manager after initialization", cm.saveSync())
}

// load will pull all saved state from the contract manager off of the disk and
// into memory.
func (cm *ContractManager) load() error {
	// Load the most recent saved settings into the contract manager.
	var ss savedSettings
	err := cm.dependencies.loadFile(settingsMetadata, &ss, filepath.Join(cm.persistDir, settingsFile))
	if os.IsNotExist(err) {
		// There is no settings file, this must be the first time that the
		// contract manager has been run. Initialize the contracter with
		// default settings.
		return cm.initSettings()
	} else if err != nil {
		return build.ExtendErr("error loading the contract manager settings file", err)
	}

	// Copy the saved settings into the contract manager.
	cm.sectorSalt = ss.SectorSalt
	cm.storageFolders = ss.StorageFolders

	// TODO: Load the sector locations from the various storage folders.

	//  Load any uncommitted changes that were recorded by the WAL.
	return build.ExtendErr("error loading the contract manager WAL", cm.wal.load())
}

// save will commit the in-memory contract manager data to disk.
func (cm *ContractManager) saveSync() error {
	ss := cm.savedSettings()
	return build.ExtendErr("error saving contract manager settings", persist.SaveFileSync(settingsMetadata, &ss, filepath.Join(cm.persistDir, settingsFile)))
}

// savedSettings returns the settings of the contract manager in an
// easily-serializable form.
func (cm *ContractManager) savedSettings() savedSettings {
	return savedSettings{
		SectorSalt:     cm.sectorSalt,
		StorageFolders: cm.storageFolders,
	}
}