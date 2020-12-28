.PHONY: install

all: kereru_app kereru_bot kereru_setup

kereru_app:
	CGO_ENABLED=0 go build -trimpath kereru_app.go config.go common.go tweet.go rbac.go image.go video.go user.go helper.go

kereru_bot:
	CGO_ENABLED=0 go build -trimpath kereru_bot.go config.go common.go

kereru_setup:
	CGO_ENABLED=0 go build -trimpath kereru_setup.go config.go

install:
	mkdir -p "$(DESTDIR)/usr/share/kereru/schema"
	cp -R "schema/mysql" "$(DESTDIR)/usr/share/kereru/schema/mysql"
	install -D "etc/kereru/default.yml" -m644  "$(DESTDIR)/etc/kereru/default.yml"
	install -D "etc/kereru/config.yml" -m644  "$(DESTDIR)/etc/kereru/config.yml"
	mkdir -p "$(DESTDIR)/usr/share/kereru"
	mkdir -p "$(DESTDIR)/var/kereru/uploads/thumbs"
	cp -R "css"  "$(DESTDIR)/usr/share/kereru/css"
	cp -R "js" "$(DESTDIR)/usr/share/kereru/js"
	cp -R "templates" "$(DESTDIR)/usr/share/kereru/templates"
	install -D "kereru_setup" -m755  "$(DESTDIR)/usr/bin/kereru_setup"
	install -D "kereru_app" -m755  "$(DESTDIR)/usr/bin/kereru_app"
	install -D "kereru_bot" -m755  "$(DESTDIR)/usr/bin/kereru_bot"


