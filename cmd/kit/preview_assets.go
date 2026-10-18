package main

import (
	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/gameinstall"
	"aoe2kit/pkg/preview"
	"aoe2kit/pkg/scenario"
)

func runPreviewAssetFetch(a artworkStudyArgs) {
	source, err := gameinstall.Resolve(a.from)
	if err != nil {
		die("kit preview assets", err)
	}
	if a.dat == "" {
		a.dat = gameinstall.DataPath(source)
	}
	idx, err := datfile.Open(a.dat)
	if err != nil {
		die("kit preview assets", err)
	}
	art, err := scenario.OpenArtwork(a.input)
	if err != nil {
		die("kit preview assets", err)
	}
	manifest, err := preview.ArtworkAssets(art, idx, a.to, a.options.Crop)
	if err != nil {
		die("kit preview assets", err)
	}
	report, err := preview.FetchMissingSLDs(manifest, source, a.to)
	printJSON(report)
	if err != nil {
		die("kit preview assets", err)
	}
}
