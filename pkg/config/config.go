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

package config

import (
	"path/filepath"

	"github.com/rancher-sandbox/elemental/pkg/cloudinit"
	"github.com/rancher-sandbox/elemental/pkg/constants"
	"github.com/rancher-sandbox/elemental/pkg/http"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
	"github.com/twpayne/go-vfs"
	"k8s.io/mount-utils"
)

type GenericOptions func(a *v1.Config) error

func WithFs(fs v1.FS) func(r *v1.Config) error {
	return func(r *v1.Config) error {
		r.Fs = fs
		return nil
	}
}

func WithLogger(logger v1.Logger) func(r *v1.Config) error {
	return func(r *v1.Config) error {
		r.Logger = logger
		return nil
	}
}

func WithSyscall(syscall v1.SyscallInterface) func(r *v1.Config) error {
	return func(r *v1.Config) error {
		r.Syscall = syscall
		return nil
	}
}

func WithMounter(mounter mount.Interface) func(r *v1.Config) error {
	return func(r *v1.Config) error {
		r.Mounter = mounter
		return nil
	}
}

func WithRunner(runner v1.Runner) func(r *v1.Config) error {
	return func(r *v1.Config) error {
		r.Runner = runner
		return nil
	}
}

func WithClient(client v1.HTTPClient) func(r *v1.Config) error {
	return func(r *v1.Config) error {
		r.Client = client
		return nil
	}
}

func WithCloudInitRunner(ci v1.CloudInitRunner) func(r *v1.Config) error {
	return func(r *v1.Config) error {
		r.CloudInitRunner = ci
		return nil
	}
}

func WithLuet(luet v1.LuetInterface) func(r *v1.Config) error {
	return func(r *v1.Config) error {
		r.Luet = luet
		return nil
	}
}

func WithArch(arch string) func(r *v1.Config) error {
	return func(r *v1.Config) error {
		r.Arch = arch
		return nil
	}
}

func NewConfig(opts ...GenericOptions) *v1.Config {
	log := v1.NewLogger()
	c := &v1.Config{
		Fs:      vfs.OSFS,
		Logger:  log,
		Syscall: &v1.RealSyscall{},
		Client:  http.NewClient(),
		Repos:   []v1.Repository{},
		Arch:    "x86_64",
	}
	for _, o := range opts {
		err := o(c)
		if err != nil {
			return nil
		}
	}

	// delay runner creation after we have run over the options in case we use WithRunner
	if c.Runner == nil {
		c.Runner = &v1.RealRunner{Logger: c.Logger}
	}

	// Now check if the runner has a logger inside, otherwise point our logger into it
	// This can happen if we set the WithRunner option as that doesn't set a logger
	if c.Runner.GetLogger() == nil {
		c.Runner.SetLogger(c.Logger)
	}

	// Delay the yip runner creation, so we set the proper logger instead of blindly setting it to the logger we create
	// at the start of NewRunConfig, as WithLogger can be passed on init, and that would result in 2 different logger
	// instances, on the config.Logger and the other on config.CloudInitRunner
	if c.CloudInitRunner == nil {
		c.CloudInitRunner = cloudinit.NewYipCloudInitRunner(c.Logger, c.Runner, vfs.OSFS)
	}

	if c.Mounter == nil {
		c.Mounter = mount.New(constants.MountBinary)
	}
	return c
}

func NewRunConfig(opts ...GenericOptions) *v1.RunConfig {
	r := &v1.RunConfig{
		Config: *NewConfig(opts...),
	}
	// Set defaults if empty
	if r.GrubConf == "" {
		r.GrubConf = constants.GrubConf
	}

	if r.ActiveLabel == "" {
		r.ActiveLabel = constants.ActiveLabel
	}

	if r.PassiveLabel == "" {
		r.PassiveLabel = constants.PassiveLabel
	}

	if r.SystemLabel == "" {
		r.SystemLabel = constants.SystemLabel
	}

	if r.RecoveryLabel == "" {
		r.RecoveryLabel = constants.RecoveryLabel
	}

	if r.PersistentLabel == "" {
		r.PersistentLabel = constants.PersistentLabel
	}

	if r.OEMLabel == "" {
		r.OEMLabel = constants.OEMLabel
	}

	if r.StateLabel == "" {
		r.StateLabel = constants.StateLabel
	}

	r.Partitions = v1.PartitionList{}
	r.Images = v1.ImageMap{}

	if r.GrubDefEntry == "" {
		r.GrubDefEntry = constants.GrubDefEntry
	}

	if r.ImgSize == 0 {
		r.ImgSize = constants.ImgSize
	}
	return r
}

func NewRunConfigNew(opts ...GenericOptions) *v1.RunConfigNew {
	config := NewConfig(opts...)
	r := &v1.RunConfigNew{
		Config:  *config,
		Install: *NewInstallSpec(*config),
	}
	return r
}

// NewInstallSpec returns an InstallSpec struct all based on defaults and basic host checks (e.g. EFI vs BIOS)
func NewInstallSpec(cfg v1.Config) *v1.InstallSpec {
	var firmware string
	var recoveryImg, activeImg, passiveImg v1.Image

	recoveryImgFile := filepath.Join(constants.IsoMnt, constants.RecoverySquashFile)

	// Check if current host has EFI firmware
	efiExists, _ := utils.Exists(cfg.Fs, constants.EfiDevice)
	// Check the default ISO installation media is available
	isoRootExists, _ := utils.Exists(cfg.Fs, constants.IsoBaseTree)
	// Check the default ISO recovery installation media is available)
	recoveryExists, _ := utils.Exists(cfg.Fs, recoveryImgFile)

	if efiExists {
		firmware = v1.EFI
	} else {
		firmware = v1.BIOS
	}

	activeImg.Label = constants.ActiveLabel
	activeImg.Size = constants.ImgSize
	activeImg.File = filepath.Join(constants.StateDir, "cOS", constants.ActiveImgFile)
	activeImg.FS = constants.LinuxImgFs
	activeImg.MountPoint = constants.ActiveDir
	if isoRootExists {
		activeImg.Source = v1.NewDirSrc(constants.IsoBaseTree)
	} else {
		activeImg.Source = v1.NewEmptySrc()
	}

	recoveryImg.File = filepath.Join(constants.RecoveryDir, "cOS", constants.RecoverySquashFile)
	if recoveryExists {
		recoveryImg.Source = v1.NewFileSrc(recoveryImgFile)
		recoveryImg.FS = constants.SquashFs
	} else {
		recoveryImg.Source = v1.NewFileSrc(activeImg.File)
		recoveryImg.FS = constants.LinuxImgFs
		recoveryImg.Label = constants.SystemLabel
	}

	passiveImg = v1.Image{
		File:   filepath.Join(constants.StateDir, "cOS", constants.PassiveImgFile),
		Label:  constants.PassiveLabel,
		Source: v1.NewFileSrc(activeImg.File),
		FS:     constants.LinuxImgFs,
	}

	return &v1.InstallSpec{
		Firmware:    firmware,
		PartTable:   v1.GPT,
		Partitions:  NewInstallParitionMap(),
		ActiveImg:   activeImg,
		RecoveryImg: recoveryImg,
		PassiveImg:  passiveImg,
	}
}

func NewInstallParitionMap() v1.PartitionMap {
	partitions := v1.PartitionMap{}
	partitions[constants.OEMPartName] = &v1.Partition{
		Label:      constants.OEMLabel,
		Size:       constants.OEMSize,
		Name:       constants.OEMPartName,
		FS:         constants.LinuxFs,
		MountPoint: constants.OEMDir,
		Flags:      []string{},
	}

	partitions[constants.RecoveryPartName] = &v1.Partition{
		Label:      constants.RecoveryLabel,
		Size:       constants.RecoverySize,
		Name:       constants.RecoveryPartName,
		FS:         constants.LinuxFs,
		MountPoint: constants.RecoveryDir,
		Flags:      []string{},
	}

	partitions[constants.StatePartName] = &v1.Partition{
		Label:      constants.StateLabel,
		Size:       constants.StateSize,
		Name:       constants.StatePartName,
		FS:         constants.LinuxFs,
		MountPoint: constants.StateDir,
		Flags:      []string{},
	}

	partitions[constants.PersistentPartName] = &v1.Partition{
		Label:      constants.PersistentLabel,
		Size:       constants.PersistentSize,
		Name:       constants.PersistentPartName,
		FS:         constants.LinuxFs,
		MountPoint: constants.PersistentDir,
		Flags:      []string{},
	}
	return partitions
}

func NewISO() *v1.LiveISO {
	return &v1.LiveISO{
		Label:       constants.ISOLabel,
		UEFI:        constants.GetDefaultISOUEFI(),
		Image:       constants.GetDefaultISOImage(),
		HybridMBR:   constants.IsoHybridMBR,
		BootFile:    constants.IsoBootFile,
		BootCatalog: constants.IsoBootCatalog,
	}
}

func NewBuildConfig(opts ...GenericOptions) *v1.BuildConfig {
	b := &v1.BuildConfig{
		Config: *NewConfig(opts...),
		ISO:    NewISO(),
		Name:   constants.BuildImgName,
	}
	return b
}
