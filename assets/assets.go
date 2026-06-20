package assets

import "embed"

// FS contains generated pixel sprites and tray icons used by the Windows app.
//
//go:embed sprites/*.png tray.ico
var FS embed.FS
