package main

import (
	"context"

	"github.com/go-logr/logr"
	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgconvco"
	"go.einride.tech/sage/tools/sggit"
	"go.einride.tech/sage/tools/sggo"
	"go.einride.tech/sage/tools/sggolangcilint"
	"go.einride.tech/sage/tools/sggoreview"
	"go.einride.tech/sage/tools/sgmarkdownfmt"
	"go.einride.tech/sage/tools/sgyamlfmt"
)

func main() {
	sg.GenerateMakefiles(
		sg.Makefile{
			Path:          sg.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
	)
}

func All(ctx context.Context) error {
	sg.Deps(ctx, ConvcoCheck, GolangciLint, GoReview, GoTest, FormatMarkdown, FormatYAML, GoGenerate)
	sg.SerialDeps(ctx, GoModTidy, GitVerifyNoDiff)
	return nil
}

func FormatYAML(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("formatting YAML files...")
	return sgyamlfmt.FormatYAML(ctx)
}

func GoModTidy(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("tidying Go module files...")
	return sg.Command(ctx, "go", "mod", "tidy", "-v").Run()
}

func GoTest(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("running Go tests...")
	return sggo.TestCommand(ctx).Run()
}

func GoReview(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("reviewing Go files...")
	return sggoreview.Command(ctx, "-c", "1", "./...").Run()
}

func GolangciLint(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("linting Go files...")
	return sggolangcilint.Run(ctx)
}

func FormatMarkdown(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("formatting Markdown files...")
	return sgmarkdownfmt.Command(ctx, "-w", ".").Run()
}

func ConvcoCheck(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("checking git commits...")
	return sgconvco.Command(ctx, "check", "origin/master..HEAD").Run()
}

func GitVerifyNoDiff(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("verifying that git has no diff...")
	return sggit.VerifyNoDiff(ctx)
}

func Stringer(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("building...")
	_, err := sgtool.GoInstall(ctx, "golang.org/x/tools/cmd/stringer", "v0.1.8")
	return err
}

func GoGenerate(ctx context.Context) error {
	sg.Deps(ctx, Stringer)
	logr.FromContextOrDiscard(ctx).Info("generating Go code...")
	return sg.Command(ctx, "go", "generate", "./...").Run()
}
