#!/bin/sh
export GOFLAGS="-trimpath"

go build kereru_app.go config.go common.go tweet.go rbac.go image.go video.go user.go helper.go

go build kereru_bot.go config.go common.go

go build kereru_setup.go config.go

exit 0
