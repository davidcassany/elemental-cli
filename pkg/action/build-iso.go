/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package action

import (
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
)

// BuildISORun will install the system from a given configuration
func BuildISORun(cfg *v1.BuildConfig) (err error) {
	cleanup := utils.NewCleanStack()
	defer func() { err = cleanup.Cleanup(err) }()

	rootDir, err := utils.TempDir(cfg.Fs, "", "elemental-iso-rootfs")
	if err != nil {
		return err
	}
	cleanup.Push(func() error { return cfg.Fs.RemoveAll(rootDir) })

	uefiDir, err := utils.TempDir(cfg.Fs, "", "elemental-iso-uefi")
	if err != nil {
		return err
	}
	cleanup.Push(func() error { return cfg.Fs.RemoveAll(uefiDir) })

	isoDir, err := utils.TempDir(cfg.Fs, "", "elemental-iso-iso")
	if err != nil {
		return err
	}
	cleanup.Push(func() error { return cfg.Fs.RemoveAll(isoDir) })

	err = prepareRootfs(cfg.Config, rootDir, cfg.ISO.RootFS...)
	if err != nil {
		return err
	}

	return err
}

func prepareRootfs(c v1.Config, root string, sources ...string) error {
	for _, src := range sources {
		c.Logger.Debugf("Attempting to apply %s at %s", src, root)
	}
	return nil
}
