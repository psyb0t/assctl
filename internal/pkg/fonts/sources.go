package fonts

import (
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	dejaVuArchiveURL    = "https://github.com/dejavu-fonts/dejavu-fonts/releases/download/version_2_37/dejavu-fonts-ttf-2.37.zip"
	dejaVuArchiveSHA    = "7576310b219e04159d35ff61dd4a4ec4cdba4f35c00e002a136f00e96a908b0a"
	dejaVuArchivePrefix = "dejavu-fonts-ttf-2.37/"
	dejaVuFontPrefix    = dejaVuArchivePrefix + "ttf/"
	googleFontsBase     = "https://raw.githubusercontent.com/google/fonts/5026800714d07231937d92eb4cceabf06e05d74e/ofl/"
	tiktokArchiveURL    = "https://github.com/tiktok/TikTokSans/releases/download/v4.000/TikTokSans-v4.000.zip"
	tiktokArchiveSHA    = "b238e0e794735142896025ea1f091a1a9ca49a9b26ca77ab5dc2ef5681b80243"
	theBoldArchiveURL   = "https://dl.dafont.com/dl/?f=the_bold_font"
	theBoldArchiveSHA   = "fd1bdadf7a4e4280412eb90e73278006fcdd933f3ddbfe315ce416ed83718951"
)

// fontAsset is one installed file. A file inside a downloaded zip archive names
// the archive in url and checksum, its path in member, and its own hash in fileChecksum.
type fontAsset struct {
	name         string
	url          string
	checksum     string
	member       string
	fileChecksum string
}

// digest is the SHA-256 of the installed file.
func (a fontAsset) digest() string {
	if a.fileChecksum != "" {
		return a.fileChecksum
	}

	return a.checksum
}

// fontSource is a font file and the license or notice installed beside it.
type fontSource struct {
	font    fontAsset
	license fontAsset
}

// catalog maps font families to their sources. A family in bold uses its bold
// source when the preset is bold and the regular source otherwise.
type catalog struct {
	regular map[string]fontSource
	bold    map[string]fontSource
}

func (c catalog) source(preset ass.Preset) (fontSource, error) {
	if source, ok := c.bold[preset.Font]; ok && preset.Bold {
		return source, nil
	}
	if source, ok := c.regular[preset.Font]; ok {
		return source, nil
	}

	return fontSource{}, ctxerrors.Wrapf(commerr.ErrUnsupported, "no bundled source for font %q", preset.Font)
}

func dejaVuSource(file, fileChecksum string) fontSource {
	return fontSource{
		font: fontAsset{
			name: file, url: dejaVuArchiveURL, checksum: dejaVuArchiveSHA,
			member: dejaVuFontPrefix + file, fileChecksum: fileChecksum,
		},
		license: fontAsset{
			name: "LICENSE", url: dejaVuArchiveURL, checksum: dejaVuArchiveSHA,
			member:       dejaVuArchivePrefix + "LICENSE",
			fileChecksum: "7a083b136e64d064794c3419751e5c7dd10d2f64c108fe5ba161eae5e5958a93",
		},
	}
}

var builtInFonts = catalog{
	regular: map[string]fontSource{
		ass.FontDejaVuSans: dejaVuSource(
			"DejaVuSans.ttf", "7da195a74c55bef988d0d48f9508bd5d849425c1770dba5d7bfc6ce9ed848954",
		),
		ass.FontDejaVuSerif: dejaVuSource(
			"DejaVuSerif.ttf", "42d1edeb7952f31b1f96d767ed7030b08a39e0c372b0071641518864e2bffb51",
		),
		// The only built-in mono preset is bold, so the bold file is the one installed.
		ass.FontDejaVuSansMono: dejaVuSource(
			"DejaVuSansMono-Bold.ttf", "bce60f1b4421acd9ea51ba6623d7024ecbe6817a953e3654df62a5e6bdf8f769",
		),
		ass.FontTheBold: {
			font: fontAsset{
				name: "THEBOLDFONT-FREEVERSION.ttf", url: theBoldArchiveURL, checksum: theBoldArchiveSHA,
				member:       "THEBOLDFONT-FREEVERSION.ttf",
				fileChecksum: "5751022bdfe00903cf6ec5d4a8d23920952ecc213169bbaf19f28d8dedc54339",
			},
			license: fontAsset{
				name: "TheBoldFont-LICENSE.rtf", url: theBoldArchiveURL, checksum: theBoldArchiveSHA,
				member:       "THE BOLD FONT - LICENSE.rtf",
				fileChecksum: "d0af67caa8e3db016bcb09f3cdfbded0ea18d44729e71508d9d60bf2b3eedf18",
			},
		},
		ass.FontTikTokSans: {
			font: fontAsset{
				name: "TikTokSans36pt-Bold.ttf", url: tiktokArchiveURL, checksum: tiktokArchiveSHA,
				member:       "TikTokSans-v4.000/fonts/ttf/TikTokSans36pt-Bold.ttf",
				fileChecksum: "f6da258a8664b99d1a5a071cc7aebedd06e4acae6e97b8f18df7af640089636f",
			},
			license: fontAsset{
				name: "TikTokSans-OFL.txt", url: tiktokArchiveURL, checksum: tiktokArchiveSHA,
				member:       "TikTokSans-v4.000/OFL.txt",
				fileChecksum: "99e743f68f5eac487004c73aaca69c5d80d91ddb7b75c04bdf5f39331d9b7158",
			},
		},
		ass.FontAnton: {
			font: fontAsset{
				name: "Anton-Regular.ttf", url: googleFontsBase + "anton/Anton-Regular.ttf",
				checksum: "a4ba3a92350ebb031da0cb47630ac49eb265082ca1bc0450442f4a83ab947cab",
			},
			license: fontAsset{
				name: "Anton-OFL.txt", url: googleFontsBase + "anton/OFL.txt",
				checksum: "ee67e6ee22790b7929f1a3769ca2801d565c64b5a9096942c1adf5596de9c9e4",
			},
		},
		ass.FontBarlowCondensedExtraBold: {
			font: fontAsset{
				name: "BarlowCondensed-ExtraBold.ttf", url: googleFontsBase + "barlowcondensed/BarlowCondensed-ExtraBold.ttf",
				checksum: "724c9c25952d5f4a2d87185d9767aa006144c5f0d944dc05bf7d5d603551c260",
			},
			license: fontAsset{
				name: "BarlowCondensed-OFL.txt", url: googleFontsBase + "barlowcondensed/OFL.txt",
				checksum: "186d750eb496a4c17a76385f82be6aea2ac1cf2de074a811d63786cf374ea73f",
			},
		},
	},
	bold: map[string]fontSource{
		ass.FontDejaVuSans: dejaVuSource(
			"DejaVuSans-Bold.ttf", "e6476c1b80502924294eed40894c5b18e06c181444ca953e5334262df9c27724",
		),
	},
}
