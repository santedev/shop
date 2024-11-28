PORT ?= 8000
run: build
	@./bin/app
build:
	@~/go/bin/templ generate && \
	 npx tailwindcss -i tailwind/css/app.css -o public/styles.css && \
		 go build -o bin/app .
tailwind:
	@npx tailwindcss -i tailwind/css/app.css -o public/styles/styles.css --watch
css:
	@npx tailwindcss -i tailwind/css/app.css -o public/styles/styles.css --watch
templ:
	~/go/bin/templ generate --watch
templ-proxy:
	~/go/bin/templ generate --watch --proxy=http://localhost:$(PORT)
get-dependecies:
	@curl -sLo public/scripts/htmx.min.js https://cdn.jsdelivr.net/npm/htmx.org/dist/htmx.min.js && \
	curl -sLo public/scripts/hyperscript.min.js https://unpkg.com/hyperscript.org/dist/_hyperscript.min.js && \
	curl -sLo public/scripts/alpine.min.js https://cdn.jsdelivr.net/npm/alpinejs/dist/cdn.min.js && \
	curl -sLo public/scripts/alpine.focus.min.js https://cdn.jsdelivr.net/npm/@alpinejs/focus/dist/cdn.min.js && \
	curl -sLo public/scripts/jquery.min.js https://cdn.jsdelivr.net/npm/jquery/dist/jquery.min.js
bundle-all: build-js build-css
	@echo "Bundling complete!"
build-js:
	@npx esbuild public/modules/main.js --minify --outfile=public/scripts/bundle.min.js
build-css:
	@npx esbuild public/styles/main.css --minify --outfile=public/styles/bundle.min.css
npm-pkg:
	@npm install -D tailwindcss esbuild
