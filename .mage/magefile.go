//go:build mage
// +build mage

package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgmake"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgyamlfmt"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgconvco"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggo"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggoreview"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggolangcilint"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgmarkdownfmt"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggitverifynodiff"
)

func init() {
	mgmake.GenerateMakefiles(
		mgmake.Makefile{
			Path:          mgpath.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
	)
}

func All() {
	mg.Deps(
		mg.F(mgconvco.ConvcoCheck, "origin/master..HEAD"),
		mggolangcilint.GolangciLint,
		mggoreview.Goreview,
		mggo.GoTest,
		mgmarkdownfmt.FormatMarkdown,
		mgyamlfmt.FormatYaml,
	)
	mg.SerialDeps(
		mggo.GoModTidy,
		mggitverifynodiff.GitVerifyNoDiff,
	)
}

func GoGenerate(ctx context.Context) error {
	stringer, err := mgtool.GoInstall(ctx, "golang.org/x/tools/cmd/stringer", "v0.1.8")
	if err != nil {
		return err
	}
	mglog.Logger("go-generate").Info("generating...")
	return sh.RunWithV(
		map[string]string{"PATH": filepath.Dir(stringer) + ":" + os.Getenv("PATH")},
		"go",
		"generate",
		"./...",
	)
}
