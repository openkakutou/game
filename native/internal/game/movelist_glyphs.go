package game

// Ikemen GO's glyph tokens (src/resources/defaultMotif.ini [Glyphs], lines
// 6477-6559 at commit 54d4148a728153c2097a0fb899a06812b2d3e193), 82 keys,
// each before any key it starts (_DF before _D), as external/script/menu.lua:780
// tries them longest first.
var glyphKeys = []string{
	"_COUNTER", "_RELEASE", "_CHARGE", "_AIROK", "_THROW", "_CLOSE", "_MASH", "_HOLD", "_FAR",
	"_HCB", "_HUF", "_HCF", "_HUB", "_QFD", "_QDB", "_QCB", "_QBU", "_QUF", "_QBD", "_QDF", "_QCF", "_QFU", "_QUB",
	"_FDF", "_FUB", "_FUF", "_FDB", "_XFF", "_XBB", "_DSF", "_DSB", "_AIR", "_TAP", "_MAX",
	"_EX", "_DB", "_DF", "_UB", "_UF",
	"^LP", "^MP", "^HP", "^LK", "^MK", "^HK", "^3K", "^3P", "^2K", "^2P",
	"~DB", "~DF", "~UB", "~UF",
	"~D", "~B", "~F", "~U",
	"^A", "^B", "^C", "^D", "^W", "^X", "^Y", "^Z", "^S", "^M", "^P", "^K",
	"_+", "_.", "_D", "_B", "_F", "_U", "_-", "_!", "_^", "_=", "_)", "_(", "_`",
}

// glyphIcons are the tokens that stand for one of the screenpack's button
// icons; the others are drawn as text. The buttons' names are the ones in the
// table's comments: [Glyphs] lines 6478-6485 (^A..^D, ^W..^Z are the letters
// of the buttons), 6489-6494 (_D Down, _B Back, _F Forward, _U Up), 6496 (^S
// Start), 6500-6505 (^LP Light Punch (X), ^MP (Y), ^HP (Z), ^LK Light Kick
// (A), ^MK (B), ^HK (C)).
var glyphIcons = map[string]string{
	"^A": "a", "^B": "b", "^C": "c", "^X": "x", "^Y": "y", "^Z": "z", "^S": "start",
	"_U": "up", "_D": "down", "_F": "forward", "_B": "back",
	"^LP": "x", "^MP": "y", "^HP": "z", "^LK": "a", "^MK": "b", "^HK": "c",
}
