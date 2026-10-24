//go:build uiharness

package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	workbookName   = "client-workbook.csv"
	workbookFile   = 0o600
	duplicateTitle = "Breville Bambino Plus review: a year on the counter"
	plannedTitle   = "Descaling guide for lever machines"
	misplacedPath  = "/grinders/hand/"
)

func workbook() string {
	return "URL,Title,Category\n" +
		"," + duplicateTitle + ",\n" +
		"," + plannedTitle + ",Espresso machines\n" +
		misplacedPath + ",Hand grinders that can actually grind for espresso,Brewing technique\n"
}

func workbookMapping(siteID string) imports.Mapping {
	return imports.Mapping{
		SiteID: siteID, Name: "Client workbook",
		Columns: map[string]string{string(importmap.FieldPath): "URL", string(importmap.FieldTitle): "Title"},
		Options: imports.Options{LevelColumns: []string{"Category"}},
	}
}

func seedWorkbook(ctx context.Context, core *app.Core, siteID, home string) error {
	path := filepath.Join(home, workbookName)
	if err := os.WriteFile(path, []byte(workbook()), workbookFile); err != nil {
		return errors.New(errors.Internal, "the harness could not write the client's workbook").WithDetail("path", path).WithInternal(err)
	}
	_, err := core.Imports.Apply(ctx, imports.ApplyRequest{SiteID: siteID, Path: path, Mapping: workbookMapping(siteID)})
	return err
}
