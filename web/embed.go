package web

import "embed"

// DistFS 嵌入的前端静态文件
// 编译时会将 web/dist 目录下的所有文件打包进二进制
//
//go:embed dist/*
var DistFS embed.FS
